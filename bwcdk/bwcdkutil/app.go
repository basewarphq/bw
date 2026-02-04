package bwcdkutil

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

// SharedConstructor creates shared infrastructure in a given stack.
// It returns the shared construct that will be passed to deployment constructors.
type SharedConstructor[S any] func(stack awscdk.Stack) S

// DeploymentConstructor creates deployment-specific infrastructure in a given stack.
// It receives the deployment identifier. Resources from shared stacks should be
// looked up via SSM Parameter Store to avoid cross-stack references.
type DeploymentConstructor func(stack awscdk.Stack, deploymentIdent string)

// AppConfig configures the CDK app setup.
type AppConfig struct {
	// Prefix for context keys (e.g., "myapp-" for "myapp-qualifier", "myapp-primary-region", etc.)
	Prefix string
}

// SetupApp configures a CDK app with multi-region, multi-deployment stacks.
//
// It creates:
//  1. A primary shared stack using the SharedConstructor
//  2. Secondary shared stacks for each secondary region (dependent on primary)
//  3. Deployment stacks for each allowed deployment in the primary region
//  4. Secondary deployment stacks for each secondary region (dependent on primary deployment)
//
// The type parameter S represents the shared construct type returned by SharedConstructor.
// SetupApp validates all context values upfront and panics with a clear error message
// if any required values are missing or invalid.
func SetupApp[S any](
	app awscdk.App,
	cfg AppConfig,
	newShared SharedConstructor[S],
	newDeployment DeploymentConstructor,
) {
	// Validate all context values upfront and store in construct tree
	config, err := NewConfig(app, cfg)
	if err != nil {
		panic(err)
	}
	StoreConfig(app, config)

	// Create shared primary region stack first.
	primarySharedStack := NewStackFromConfig(app, config, config.PrimaryRegion)
	_ = newShared(primarySharedStack)

	// Create secondary shared region stacks with dependency on primary.
	// Secondary shared stacks reference resources (like Route53 hosted zone IDs)
	// stored by the primary shared stack, so they must deploy after it.
	secondarySharedStacks := make([]awscdk.Stack, 0, len(config.SecondaryRegions))
	for _, region := range config.SecondaryRegions {
		secondarySharedStack := NewStackFromConfig(app, config, region)
		_ = newShared(secondarySharedStack)
		secondarySharedStack.AddDependency(primarySharedStack, jsii.String("Primary region must deploy first"))
		secondarySharedStacks = append(secondarySharedStacks, secondarySharedStack)
	}

	// Create stacks for each deployment.
	for _, deploymentIdent := range config.Deployments {
		primaryDeploymentStack := NewStackFromConfig(app, config, config.PrimaryRegion, deploymentIdent)
		newDeployment(primaryDeploymentStack, deploymentIdent)

		// The primary deployment stack depends on ALL shared stacks (primary and secondary).
		// This ensures all shared infrastructure is fully provisioned across all regions
		// before any deployment begins. This creates a clean two-phase deployment:
		//   Phase 1: All shared stacks (primary → secondary)
		//   Phase 2: All deployment stacks (primary → secondary)
		// This simplifies reasoning about deployment order and ensures secondary deployments
		// can reference resources from their regional shared stacks.
		primaryDeploymentStack.AddDependency(primarySharedStack,
			jsii.String("All shared stacks must deploy before deployments"))
		for _, secondarySharedStack := range secondarySharedStacks {
			primaryDeploymentStack.AddDependency(secondarySharedStack,
				jsii.String("All shared stacks must deploy before deployments"))
		}

		// Secondary deployment stacks depend on the primary deployment stack.
		for _, region := range config.SecondaryRegions {
			secondaryDeploymentStack := NewStackFromConfig(app, config, region, deploymentIdent)
			newDeployment(secondaryDeploymentStack, deploymentIdent)
			secondaryDeploymentStack.AddDependency(primaryDeploymentStack,
				jsii.String("Primary region deployment must deploy first"))
		}
	}
}
