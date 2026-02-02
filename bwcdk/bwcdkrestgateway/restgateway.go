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
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdklwalambda"
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
}

type restGateway struct {
	lambda         bwcdklwalambda.Lambda
	restApi        awsapigateway.RestApi
	accessLogGroup awslogs.ILogGroup
}

// New creates a RestGateway construct with a Lambda-backed REST API.
//
// Only paths specified in PublicRoutes are accessible externally.
// The Lambda can expose additional internal paths (e.g., /lambda/*)
// that remain accessible only via direct Lambda invocation.
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

	con.restApi = awsapigateway.NewRestApi(scope, jsii.String("Api"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String(apiName),
		DeployOptions: &awsapigateway.StageOptions{
			StageName: jsii.String("prod"),
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
