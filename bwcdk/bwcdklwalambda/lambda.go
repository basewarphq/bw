// Package bwcdklwalambda provides a reusable Lambda construct for Go functions
// using AWS Lambda Web Adapter (LWA) for HTTP-based handlers.
//
// The construct handles Go bundling with reproducible builds and configures
// the Lambda Web Adapter layer automatically. Functions run an HTTP server
// that LWA forwards Lambda invocations to.
package bwcdklwalambda

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkloggroup"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
	"github.com/cockroachdb/errors"
	"github.com/iancoleman/strcase"
)

// LWALayerVersion is the current version of the Lambda Web Adapter layer.
const LWALayerVersion = 25

// Lambda provides access to a Go Lambda function with AWS Lambda Web Adapter.
type Lambda interface {
	// Function returns the underlying Lambda function.
	Function() awscdklambdagoalpha.GoFunction
	// LogGroup returns the CloudWatch Log Group for the function.
	LogGroup() awslogs.ILogGroup
	// Name returns the construct name derived from the entry path.
	Name() string
}

// Props configures the Lambda construct.
type Props struct {
	// Entry is the path to the Go command directory.
	// Must match pattern "<component>/cmd/<command>" (e.g., "backend/cmd/coreback").
	// The component and command are used to name the construct for AWS Console visibility.
	// Required.
	Entry *string
	// Environment variables to pass to the function.
	// PORT is set automatically for LWA.
	Environment *map[string]*string
	// PassThroughPath sets AWS_LWA_PASS_THROUGH_PATH for non-HTTP event triggers.
	// When set, LWA POSTs the raw Lambda event JSON to this path and returns
	// the response body directly to Lambda (without HTTP wrapping).
	// Use this for Lambda authorizers and other non-HTTP triggers like SQS/SNS.
	// Optional.
	PassThroughPath *string
}

// parsePassThroughPath validates PassThroughPath and returns a suffix for construct naming.
// Path must match pattern "/l/<handler>" where handler is kebab-case.
func parsePassThroughPath(path string) (suffix string, err error) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) != 2 || parts[0] != "l" || parts[1] == "" {
		return "", errors.Newf("PassThroughPath must match pattern /l/<handler>, got %q", path)
	}
	handler := parts[1]
	if handler != strcase.ToKebab(handler) {
		return "", errors.Newf("PassThroughPath handler must be kebab-case, got %q", handler)
	}
	return strcase.ToCamel(handler), nil
}

// ParseEntry extracts component and command from entry path.
// Validates pattern "<component>/cmd/<command>".
func ParseEntry(entry string) (component, command string, err error) {
	parts := strings.Split(filepath.ToSlash(entry), "/")

	for i := len(parts) - 2; i >= 1; i-- {
		if parts[i] == "cmd" {
			component = parts[i-1]
			command = parts[i+1]
			if component == "" || command == "" {
				break
			}
			return component, command, nil
		}
	}

	return "", "", errors.Newf("entry must match pattern <component>/cmd/<command>, got %q", entry)
}

type lambda struct {
	function awscdklambdagoalpha.GoFunction
	logGroup awslogs.ILogGroup
	name     string
}

// New creates a Lambda construct with AWS Lambda Web Adapter.
//
// The function uses arm64 architecture for better price/performance and
// configures reproducible Go builds. LWA is added as a layer and configured
// to forward Lambda invocations to the HTTP server running on port 8080.
//
// The Entry path must match pattern "<component>/cmd/<command>". The component
// and command are used to name the construct (e.g., "backend/cmd/coreback" becomes
// "BackendCoreback") for better visibility in the AWS Console.
func New(scope constructs.Construct, props Props) Lambda {
	component, command, err := ParseEntry(*props.Entry)
	if err != nil {
		panic(err)
	}
	scopeName := strcase.ToCamel(component) + strcase.ToCamel(command)
	if props.PassThroughPath != nil {
		suffix, err := parsePassThroughPath(*props.PassThroughPath)
		if err != nil {
			panic(err)
		}
		scopeName += suffix
	}
	scope = constructs.NewConstruct(scope, jsii.String(scopeName))
	con := &lambda{name: scopeName}

	region := *awscdk.Stack_Of(scope).Region()

	functionName := bwcdkutil.ResourceName(scope, scopeName, bwcdkutil.CasingKebab)

	env := make(map[string]*string)
	if props.Environment != nil {
		maps.Copy(env, *props.Environment)
	}
	env["AWS_LWA_PORT"] = jsii.String("8080")
	env["AWS_LWA_READINESS_CHECK_PATH"] = jsii.String("/health")
	env["BW_SERVICE_NAME"] = jsii.String(functionName)
	env["BW_OTEL_EXPORTER"] = jsii.String("xrayudp")
	env["BW_PRIMARY_REGION"] = jsii.String(bwcdkutil.PrimaryRegion(scope))
	if props.PassThroughPath != nil {
		env["AWS_LWA_PASS_THROUGH_PATH"] = props.PassThroughPath
	}

	con.logGroup = bwcdkloggroup.New(scope, scopeName+"Logs", bwcdkloggroup.Props{
		Purpose: jsii.String("Lambda function " + scopeName),
	}).LogGroup()

	lwaLayerArn := fmt.Sprintf(
		"arn:aws:lambda:%s:753240598075:layer:LambdaAdapterLayerArm64:%d",
		region, LWALayerVersion,
	)

	con.function = awscdklambdagoalpha.NewGoFunction(scope, jsii.String("Function"),
		&awscdklambdagoalpha.GoFunctionProps{
			FunctionName: jsii.String(functionName),
			Entry:        props.Entry,
			Architecture: awslambda.Architecture_ARM_64(),
			Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
			MemorySize:   jsii.Number(128),
			Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
			Environment:  &env,
			Bundling:     bwcdkutil.ReproducibleGoBundling(),
			Tracing:      awslambda.Tracing_ACTIVE,
			Layers: &[]awslambda.ILayerVersion{
				awslambda.LayerVersion_FromLayerVersionArn(scope,
					jsii.String("LWALayer"), jsii.String(lwaLayerArn)),
			},
			LogGroup:      con.logGroup,
			LoggingFormat: awslambda.LoggingFormat_JSON,
		})

	return con
}

func (l *lambda) Function() awscdklambdagoalpha.GoFunction {
	return l.function
}

func (l *lambda) LogGroup() awslogs.ILogGroup {
	return l.logGroup
}

func (l *lambda) Name() string {
	return l.name
}
