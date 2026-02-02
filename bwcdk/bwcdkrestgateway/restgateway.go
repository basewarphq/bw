// Package bwcdkrestgateway provides a reusable REST gateway construct that wraps
// a Go Lambda function with AWS Lambda Web Adapter.
//
// The construct exposes only specified public routes, keeping internal
// Lambda paths (like /lambda/*) inaccessible from the internet.
package bwcdkrestgateway

import (
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdklwalambda"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
	"github.com/iancoleman/strcase"
)

// RestGateway provides access to a REST gateway backed by a Go Lambda function.
type RestGateway interface {
	// Lambda returns the underlying LWA Lambda construct.
	// Use this for internal integrations (SQS, EventBridge, etc.).
	Lambda() bwcdklwalambda.Lambda
	// AuthorizerLambda returns the authorizer Lambda construct, if configured.
	AuthorizerLambda() bwcdklwalambda.Lambda
	// RestApi returns the API Gateway REST API.
	RestApi() awsapigateway.RestApi
	// AccessLogGroup returns the CloudWatch Log Group for API Gateway access logs.
	AccessLogGroup() awslogs.ILogGroup
	// DomainName returns the regional custom domain name (e.g., "dev-euw1-api.basewarp.app").
	DomainName() string
	// GlobalDomainName returns the global domain name with latency-based routing (e.g., "dev-api.basewarp.app").
	GlobalDomainName() string
}

// Props configures the RestGateway construct.
type Props struct {
	// Entry is the path to the Go command directory.
	// Passed to the underlying LWA Lambda construct.
	// Required.
	Entry *string
	// PublicRoutes are the paths to expose via API Gateway.
	// Use {proxy+} for greedy path matching (e.g., "/api/{proxy+}").
	// Required.
	PublicRoutes *[]*string
	// Environment variables to pass to the Lambda function.
	Environment *map[string]*string

	// HostedZone is the Route53 hosted zone for DNS records.
	// Required.
	HostedZone awsroute53.IHostedZone
	// Certificate is the ACM certificate for the custom domain.
	// Required.
	Certificate awscertificatemanager.ICertificate
	// Subdomain is the subdomain prefix (e.g., "api").
	// Combined with deployment and region to form the full subdomain.
	// Required.
	Subdomain *string
	// DeploymentIdent is the deployment identifier (e.g., "dev", "prod").
	// Required.
	DeploymentIdent *string
	// Authorizer enables a Lambda TOKEN authorizer for all public routes.
	// When set, creates a separate Lambda instance using the same Entry,
	// configured to handle requests at /l/authorize.
	//
	// Only TOKEN authorizers are supported with AWS Lambda Web Adapter (LWA).
	// REQUEST authorizers share HTTP-like fields (httpMethod, path, headers,
	// requestContext) with proxy events, causing LWA to misroute them as
	// regular HTTP requests instead of using pass-through mode.
	//
	// Optional.
	Authorizer *AuthorizerProps
}

// AuthorizerProps configures the Lambda REQUEST authorizer.
type AuthorizerProps struct{}

type restGateway struct {
	lambda           bwcdklwalambda.Lambda
	authorizerLambda bwcdklwalambda.Lambda
	restApi          awsapigateway.RestApi
	accessLogGroup   awslogs.ILogGroup
	domainName       string
	globalDomainName string
}

// New creates a RestGateway construct with a Lambda-backed REST API.
//
// Only paths specified in PublicRoutes are accessible externally.
// The Lambda can expose additional internal paths (e.g., /lambda/*)
// that remain accessible only via direct Lambda invocation.
//
// A custom domain is configured with format "{deployment}-{region}-{subdomain}.{zone}"
// (e.g., "dev-euw1-api.basewarp.app"). The execute-api endpoint is disabled.
func New(scope constructs.Construct, props Props) RestGateway {
	scope = constructs.NewConstruct(scope, jsii.String(strcase.ToCamel(*props.Subdomain)+"RGw"))
	con := &restGateway{}

	con.lambda = bwcdklwalambda.New(scope, bwcdklwalambda.Props{
		Entry:       props.Entry,
		Environment: props.Environment,
	})

	if props.Authorizer != nil {
		con.authorizerLambda = bwcdklwalambda.New(scope, bwcdklwalambda.Props{
			Entry:       props.Entry,
			Environment: props.Environment,
			// TOKEN authorizer events don't share HTTP-like fields, so LWA
			// correctly routes them via AWS_LWA_PASS_THROUGH_PATH.
			PassThroughPath: jsii.String("/l/authorize"),
		})
	}

	apiName := con.lambda.Name() + strcase.ToCamel(*props.DeploymentIdent) + "Gateway"

	con.accessLogGroup = awslogs.NewLogGroup(scope, jsii.String("AccessLogGroup"), &awslogs.LogGroupProps{
		Retention:     awslogs.RetentionDays_ONE_WEEK,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	stack := awscdk.Stack_Of(scope)
	region := *stack.Region()
	zoneName := *props.HostedZone.ZoneName()

	regionalSubdomain := bwcdkutil.RegionalSubdomain(*props.DeploymentIdent, region, *props.Subdomain)
	con.domainName = regionalSubdomain + "." + zoneName

	globalSubdomain := bwcdkutil.GlobalSubdomain(*props.DeploymentIdent, *props.Subdomain)
	con.globalDomainName = globalSubdomain + "." + zoneName

	// Use REGIONAL endpoint type for multi-region deployments with latency-based routing.
	// Edge-optimized endpoints cannot be used with Route 53 latency-based routing because
	// the edge network requires exactly one target per domain name.
	// See: https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-api-endpoint-types.html
	con.restApi = awsapigateway.NewRestApi(scope, jsii.String("Api"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String(apiName),
		EndpointConfiguration: &awsapigateway.EndpointConfiguration{
			Types: &[]awsapigateway.EndpointType{awsapigateway.EndpointType_REGIONAL},
		},
		DomainName: &awsapigateway.DomainNameOptions{
			DomainName:   jsii.String(con.domainName),
			Certificate:  props.Certificate,
			EndpointType: awsapigateway.EndpointType_REGIONAL,
		},
		DisableExecuteApiEndpoint: jsii.Bool(true),
		DeployOptions: &awsapigateway.StageOptions{
			StageName:            jsii.String("prod"),
			AccessLogDestination: awsapigateway.NewLogGroupLogDestination(con.accessLogGroup),
			AccessLogFormat: awsapigateway.AccessLogFormat_JsonWithStandardFields(
				&awsapigateway.JsonWithStandardFieldProps{
					Caller:         jsii.Bool(true),
					HttpMethod:     jsii.Bool(true),
					Ip:             jsii.Bool(true),
					Protocol:       jsii.Bool(true),
					RequestTime:    jsii.Bool(true),
					ResourcePath:   jsii.Bool(true),
					ResponseLength: jsii.Bool(true),
					Status:         jsii.Bool(true),
					User:           jsii.Bool(true),
				}),
		},
	})

	integration := awsapigateway.NewLambdaIntegration(con.lambda.Function(), &awsapigateway.LambdaIntegrationOptions{
		Proxy: jsii.Bool(true),
	})

	var authorizer awsapigateway.IAuthorizer
	if props.Authorizer != nil {
		// Use TOKEN authorizer (not REQUEST) - REQUEST authorizers are misrouted by LWA.
		authorizer = awsapigateway.NewTokenAuthorizer(scope, jsii.String("Authorizer"),
			&awsapigateway.TokenAuthorizerProps{
				Handler:         con.authorizerLambda.Function(),
				ResultsCacheTtl: awscdk.Duration_Minutes(jsii.Number(5)),
			})
	}

	for _, route := range *props.PublicRoutes {
		addRoute(con.restApi.Root(), *route, integration, authorizer)
	}

	awsroute53.NewARecord(scope, jsii.String("DnsRecord"), &awsroute53.ARecordProps{
		Zone:       props.HostedZone,
		RecordName: jsii.String(con.domainName),
		Target:     awsroute53.RecordTarget_FromAlias(awsroute53targets.NewApiGateway(con.restApi)),
	})

	globalDomain := awsapigateway.NewDomainName(scope, jsii.String("GlobalDomain"), &awsapigateway.DomainNameProps{
		DomainName:   jsii.String(con.globalDomainName),
		Certificate:  props.Certificate,
		EndpointType: awsapigateway.EndpointType_REGIONAL,
		Mapping:      con.restApi,
	})

	awsroute53.NewARecord(scope, jsii.String("LatencyRecord"), &awsroute53.ARecordProps{
		Zone:          props.HostedZone,
		RecordName:    jsii.String(con.globalDomainName),
		Target:        awsroute53.RecordTarget_FromAlias(awsroute53targets.NewApiGatewayDomain(globalDomain)),
		Region:        stack.Region(),
		SetIdentifier: jsii.Sprintf("%s-%s", con.globalDomainName, region),
	})

	// Export endpoints as stack outputs for easy retrieval via AWS CLI.
	// Output keys include subdomain to ensure uniqueness when multiple gateways exist.
	outputPrefix := con.lambda.Name() + strcase.ToCamel(*props.Subdomain)
	awscdk.NewCfnOutput(scope, jsii.String("GatewayURLRegional"), &awscdk.CfnOutputProps{
		Key:         jsii.String(outputPrefix + "GatewayURLRegional"),
		Description: jsii.String("Regional API Gateway endpoint URL"),
		Value:       jsii.String("https://" + con.domainName),
	})
	awscdk.NewCfnOutput(scope, jsii.String("GatewayURLGlobal"), &awscdk.CfnOutputProps{
		Key:         jsii.String(outputPrefix + "GatewayURLGlobal"),
		Description: jsii.String("Global API Gateway endpoint URL (latency-based routing)"),
		Value:       jsii.String("https://" + con.globalDomainName),
	})

	return con
}

// addRoute adds a route to the REST API.
// Handles nested paths like "/api/{proxy+}" by creating intermediate resources.
func addRoute(root awsapigateway.IResource, path string, integration awsapigateway.LambdaIntegration, authorizer awsapigateway.IAuthorizer) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	resource := root
	for _, part := range parts {
		resource = resource.AddResource(jsii.String(part), nil)
	}

	var methodOpts *awsapigateway.MethodOptions
	if authorizer != nil {
		methodOpts = &awsapigateway.MethodOptions{
			Authorizer: authorizer,
		}
	}
	resource.AddMethod(jsii.String("ANY"), integration, methodOpts)
}

func (r *restGateway) Lambda() bwcdklwalambda.Lambda {
	return r.lambda
}

func (r *restGateway) AuthorizerLambda() bwcdklwalambda.Lambda {
	return r.authorizerLambda
}

func (r *restGateway) RestApi() awsapigateway.RestApi {
	return r.restApi
}

func (r *restGateway) AccessLogGroup() awslogs.ILogGroup {
	return r.accessLogGroup
}

func (r *restGateway) DomainName() string {
	return r.domainName
}

func (r *restGateway) GlobalDomainName() string {
	return r.globalDomainName
}
