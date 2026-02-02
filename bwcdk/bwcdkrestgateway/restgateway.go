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
)

// RestGateway provides access to a REST gateway backed by a Go Lambda function.
type RestGateway interface {
	// Lambda returns the underlying LWA Lambda construct.
	// Use this for internal integrations (SQS, EventBridge, etc.).
	Lambda() bwcdklwalambda.Lambda
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
}

type restGateway struct {
	lambda           bwcdklwalambda.Lambda
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
	scope = constructs.NewConstruct(scope, jsii.String("RestGateway"))
	con := &restGateway{}

	con.lambda = bwcdklwalambda.New(scope, bwcdklwalambda.Props{
		Entry:       props.Entry,
		Environment: props.Environment,
	})

	apiName := con.lambda.Name() + "Gateway"

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

	con.restApi = awsapigateway.NewRestApi(scope, jsii.String("Api"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String(apiName),
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

	for _, route := range *props.PublicRoutes {
		addRoute(con.restApi.Root(), *route, integration)
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

	return con
}

// addRoute adds a route to the REST API.
// Handles nested paths like "/api/{proxy+}" by creating intermediate resources.
func addRoute(root awsapigateway.IResource, path string, integration awsapigateway.LambdaIntegration) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	resource := root
	for _, part := range parts {
		resource = resource.AddResource(jsii.String(part), nil)
	}

	resource.AddMethod(jsii.String("ANY"), integration, nil)
}

func (r *restGateway) Lambda() bwcdklwalambda.Lambda {
	return r.lambda
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
