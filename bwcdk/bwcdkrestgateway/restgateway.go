// Package bwcdkrestgateway provides a reusable REST gateway construct that wraps
// a Go Lambda function with AWS Lambda Web Adapter.
//
// The construct exposes only specified public routes, keeping internal
// Lambda paths (like /lambda/*) inaccessible from the internet.
package bwcdkrestgateway

import (
	"iter"
	"maps"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkloggroup"
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

// RouteConfig configures a single gateway route.
type RouteConfig struct {
	// Path is the route path to expose via API Gateway.
	// Use {proxy+} for greedy path matching (e.g., "/api/{proxy+}").
	// Required.
	Path *string
	// RequireAuth specifies whether this route requires authorization.
	// Only applies when Authorizer is configured on the gateway.
	// Defaults to false if nil.
	// Optional.
	RequireAuth *bool
}

// Props configures the RestGateway construct.
type Props struct {
	// Entry is the path to the Go command directory.
	// Passed to the underlying LWA Lambda construct.
	// Required.
	Entry *string
	// GatewayRoutes are the routes to expose via API Gateway.
	// Each route can individually require authorization.
	// Required.
	GatewayRoutes *[]*RouteConfig
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
	// Authorizer enables a Lambda TOKEN authorizer for the gateway.
	// When set, creates a separate Lambda instance using the same Entry,
	// configured to handle requests at /l/authorize.
	// Routes opt-in to authorization via RouteConfig.RequireAuth.
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
// Only paths specified in GatewayRoutes are accessible externally.
// Each route can individually require authorization via RequireAuth.
// The Lambda can expose additional internal paths (e.g., /lambda/*)
// that remain accessible only via direct Lambda invocation.
//
// A custom domain is configured with format "{deployment}-{region}-{subdomain}.{zone}"
// (e.g., "dev-euw1-api.basewarp.app"). The execute-api endpoint is disabled.
func New(scope constructs.Construct, props Props) RestGateway {
	for _, route := range *props.GatewayRoutes {
		if route.RequireAuth != nil && *route.RequireAuth && props.Authorizer == nil {
			panic("route " + *route.Path + " requires auth but no Authorizer is configured")
		}
	}

	scope = constructs.NewConstruct(scope, jsii.String(strcase.ToCamel(*props.Subdomain)+"RGw"))
	con := &restGateway{}

	deploymentIdent := bwcdkutil.DeploymentIdent(scope)
	accessLogID := strcase.ToCamel(*props.Subdomain) + "AccessLogs"
	con.accessLogGroup = bwcdkloggroup.New(scope, accessLogID, bwcdkloggroup.Props{
		Purpose: jsii.String("API Gateway access logs"),
	}).LogGroup()

	// Pass the access log group name to Lambda for X-Ray log correlation.
	lambdaEnv := make(map[string]*string)
	if props.Environment != nil {
		maps.Copy(lambdaEnv, *props.Environment)
	}
	lambdaEnv["BW_GATEWAY_ACCESS_LOG_GROUP"] = con.accessLogGroup.LogGroupName()

	con.lambda = bwcdklwalambda.New(scope, bwcdklwalambda.Props{
		Entry:       props.Entry,
		Environment: &lambdaEnv,
	})

	if props.Authorizer != nil {
		con.authorizerLambda = bwcdklwalambda.New(scope, bwcdklwalambda.Props{
			Entry:       props.Entry,
			Environment: &lambdaEnv,
			// TOKEN authorizer events don't share HTTP-like fields, so LWA
			// correctly routes them via AWS_LWA_PASS_THROUGH_PATH.
			PassThroughPath: jsii.String("/l/authorize"),
		})
	}

	apiName := bwcdkutil.ResourceName(scope, con.lambda.Name()+"Gateway", bwcdkutil.CasingCamel)

	stack := awscdk.Stack_Of(scope)
	region := *stack.Region()
	zoneName := *props.HostedZone.ZoneName()

	regionalSubdomain := bwcdkutil.RegionalSubdomain(deploymentIdent, region, *props.Subdomain)
	con.domainName = regionalSubdomain + "." + zoneName

	globalSubdomain := bwcdkutil.GlobalSubdomain(deploymentIdent, *props.Subdomain)
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
			TracingEnabled:       jsii.Bool(true),
			AccessLogDestination: awsapigateway.NewLogGroupLogDestination(con.accessLogGroup),
			// Custom format to include xrayTraceId for log-to-trace correlation.
			// JsonWithStandardFields doesn't support xrayTraceId.
			AccessLogFormat: awsapigateway.AccessLogFormat_Custom(jsii.String(
				`{"requestId":"$context.requestId","ip":"$context.identity.sourceIp",` +
					`"caller":"$context.identity.caller","user":"$context.identity.user",` +
					`"requestTime":"$context.requestTime","httpMethod":"$context.httpMethod",` +
					`"resourcePath":"$context.resourcePath","status":"$context.status",` +
					`"protocol":"$context.protocol","responseLength":"$context.responseLength",` +
					`"xrayTraceId":"$context.xrayTraceId",` +
					`"integrationLatency":"$context.integration.latency",` +
					`"integrationStatus":"$context.integration.status"}`,
			)),
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

	for _, route := range *props.GatewayRoutes {
		routeAuthorizer := authorizer
		if route.RequireAuth == nil || !*route.RequireAuth {
			routeAuthorizer = nil
		}
		addRoute(con.restApi.Root(), *route.Path, integration, routeAuthorizer)
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
		Description: jsii.Sprintf("Regional %s gateway endpoint URL", *props.Subdomain),
		Value:       jsii.String("https://" + con.domainName),
	})
	awscdk.NewCfnOutput(scope, jsii.String("GatewayURLGlobal"), &awscdk.CfnOutputProps{
		Key:         jsii.String(outputPrefix + "GatewayURLGlobal"),
		Description: jsii.Sprintf("Global %s gateway endpoint URL (latency-based routing)", *props.Subdomain),
		Value:       jsii.String("https://" + con.globalDomainName),
	})

	return con
}

// addRoute adds a route to the REST API.
// Handles nested paths like "/api/{proxy+}" by creating intermediate resources.
// Root path "/" adds a method directly to the root resource.
func addRoute(
	root awsapigateway.IResource,
	path string,
	integration awsapigateway.LambdaIntegration,
	authorizer awsapigateway.IAuthorizer,
) {
	path = strings.TrimPrefix(path, "/")

	resource := root
	if path != "" {
		for part := range splitPath(path) {
			resource = resource.AddResource(jsii.String(part), nil)
		}
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

func splitPath(path string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for part := range strings.SplitSeq(path, "/") {
			if !yield(part) {
				return
			}
		}
	}
}
