// Package bwcdkloggroup provides a reusable CloudWatch Log Group construct
// with standardized retention, removal policy, and CloudFormation outputs.
//
// All log groups created with this construct automatically export their names
// as stack outputs, enabling easy discovery via AWS CLI queries.
package bwcdkloggroup

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// LogGroup provides access to a CloudWatch Log Group with standardized configuration.
type LogGroup interface {
	// LogGroup returns the underlying CDK log group.
	LogGroup() awslogs.ILogGroup
}

// Props configures the LogGroup construct.
type Props struct {
	// Purpose describes what this log group is for (e.g., "Lambda function logs").
	// Used in the CfnOutput description.
	// Required.
	Purpose *string
}

type logGroup struct {
	lg awslogs.ILogGroup
}

// New creates a LogGroup construct with standardized configuration.
//
// The log group is created with:
//   - Retention: ONE_WEEK (encapsulated, not configurable)
//   - RemovalPolicy: DESTROY (log groups are deleted with the stack)
//
// A CfnOutput is created with:
//   - Key: "{id}LogGroup" where id is derived from the construct path
//   - Value: The log group name (for CLI queries)
//   - Description: "CloudWatch Log Group for {Purpose}"
func New(scope constructs.Construct, id string, props Props) LogGroup {
	scope = constructs.NewConstruct(scope, jsii.String(id))
	con := &logGroup{}

	con.lg = awslogs.NewLogGroup(scope, jsii.String("LogGroup"), &awslogs.LogGroupProps{
		Retention:     awslogs.RetentionDays_ONE_WEEK,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	awscdk.NewCfnOutput(scope, jsii.String("LogGroupOutput"), &awscdk.CfnOutputProps{
		Key:         jsii.String(id + "LogGroup"),
		Description: jsii.String("CloudWatch Log Group for " + *props.Purpose),
		Value:       con.lg.LogGroupName(),
	})

	return con
}

func (l *logGroup) LogGroup() awslogs.ILogGroup {
	return l.lg
}
