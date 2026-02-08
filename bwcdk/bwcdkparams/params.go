// Package bwcdkparams provides utilities for storing and retrieving CDK construct
// values across AWS regions using AWS Systems Manager Parameter Store.
//
// This package enables cross-region resource sharing in multi-region CDK deployments:
//   - Primary region: Creates resources and stores identifiers in SSM Parameter Store
//   - Secondary regions: Retrieves stored values to reference existing resources
package bwcdkparams

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/aws-cdk-go/awscdk/v2/customresources"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
)

// LookupLocal retrieves a parameter from SSM Parameter Store within the same region.
// Use this for same-region cross-stack references. For cross-region lookups, use Lookup.
func LookupLocal(scope constructs.Construct, namespace string, name string) *string {
	return awsssm.StringParameter_ValueForStringParameter(scope,
		ParameterName(scope, namespace, name), nil)
}

// ParameterName generates a hierarchical SSM parameter path.
// Returns a path like /{qualifier}/{namespace}/{name}.
func ParameterName(scope constructs.Construct, namespace string, name string) *string {
	qual := bwcdkutil.Qualifier(scope)
	return jsii.Sprintf("/%s/%s/%s", qual, namespace, name)
}

// Store creates and stores a parameter in AWS SSM Parameter Store.
// Use this in the primary region to persist values for cross-region access.
func Store(scope constructs.Construct, id string, namespace string, name string, value *string) {
	awsssm.NewStringParameter(scope, jsii.String(id),
		&awsssm.StringParameterProps{
			ParameterName: ParameterName(scope, namespace, name),
			StringValue:   value,
		})
}

// Lookup retrieves a parameter stored in the primary region using a custom resource.
// Use this in secondary regions to access values created in the primary region.
// The physicalID should be a stable identifier for the custom resource (e.g., "user-pool-id-lookup").
func Lookup(scope constructs.Construct, id string, namespace string, name string, physicalID string) *string {
	sdkCall := &customresources.AwsSdkCall{
		Service: jsii.String("SSM"),
		Action:  jsii.String("getParameter"),
		Parameters: map[string]any{
			"Name": ParameterName(scope, namespace, name),
		},
		Region:             jsii.String(bwcdkutil.PrimaryRegion(scope)),
		PhysicalResourceId: customresources.PhysicalResourceId_Of(jsii.String(physicalID)),
	}
	// OnUpdate is required so that changes to the parameter path (e.g., when
	// scoping parameters per deployment) trigger a new SSM GetParameter call.
	// Without it, CloudFormation skips the SDK call on update and the response
	// is empty, causing "doesn't contain Parameter.Value" errors.
	lookup := customresources.NewAwsCustomResource(scope, jsii.String(id),
		&customresources.AwsCustomResourceProps{
			OnCreate: sdkCall,
			OnUpdate: sdkCall,
			Policy: customresources.AwsCustomResourcePolicy_FromSdkCalls(&customresources.SdkCallsPolicyOptions{
				Resources: customresources.AwsCustomResourcePolicy_ANY_RESOURCE(),
			}),
		})
	return lookup.GetResponseField(jsii.String("Parameter.Value"))
}
