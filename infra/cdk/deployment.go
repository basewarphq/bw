package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
)

func NewDeployment(stack awscdk.Stack, shared *Shared, deploymentIdent string) {
	if !shared.Base.IsValidated() {
		// Shared base not yet validated - skip deployment resources.
		return
	}

	// Add deployment-specific resources below
}
