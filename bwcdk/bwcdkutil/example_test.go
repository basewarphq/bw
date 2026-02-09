package bwcdkutil_test

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bw/bwcdk/bwcdkutil"
)

// Shared represents the shared infrastructure created once per region.
// It holds resources that are shared across all deployments in that region.
type Shared struct {
	Bucket awss3.Bucket
}

// Deployment represents deployment-specific infrastructure.
// Each deployment (Dev, Stag, Prod) gets its own instance.
type Deployment struct {
	// deployment-specific resources
}

// NewShared creates shared infrastructure in the given stack.
func NewShared(stack awscdk.Stack) *Shared {
	bucket := awss3.NewBucket(stack, jsii.String("SharedBucket"), &awss3.BucketProps{
		Versioned: jsii.Bool(true),
	})

	// Access config deep in construct tree without passing *Config explicitly
	if bwcdkutil.IsPrimaryRegion(stack, *stack.Region()) {
		// Primary region specific setup
		_ = bwcdkutil.BaseDomainName(stack)
	}

	return &Shared{Bucket: bucket}
}

// NewDeployment creates deployment-specific infrastructure.
// Resources from shared stacks should be looked up via SSM Parameter Store
// to avoid cross-stack references.
func NewDeployment(stack awscdk.Stack, deploymentIdent string) {
	_ = deploymentIdent

	// Can also get the full Config if needed
	cfg := bwcdkutil.ConfigFromScope(stack)
	_ = cfg.AllRegions()
}

// Example_setupApp demonstrates how to use SetupApp to configure a multi-region,
// multi-deployment CDK application.
//
// The cdk.json context should include:
//
//	{
//	  "myapp-qualifier": "myapp",
//	  "myapp-primary-region": "us-east-1",
//	  "myapp-secondary-regions": ["eu-west-1"],
//	  "myapp-deployments": ["Dev", "Stag", "Prod"],
//	  "myapp-base-domain-name": "example.com"
//	}
func Example_setupApp() {
	defer jsii.Close()

	ctx := map[string]any{
		"myapp-qualifier":         "myapp",
		"myapp-primary-region":    "us-east-1",
		"myapp-secondary-regions": []any{"eu-west-1"},
		"myapp-deployments":       []any{"Dev", "Stag", "Prod"},
		"myapp-base-domain-name":  "example.com",
	}

	app := awscdk.NewApp(&awscdk.AppProps{
		Context: &ctx,
	})

	bwcdkutil.SetupApp(app, bwcdkutil.AppConfig{
		Prefix: "myapp-",
	},
		NewShared,
		NewDeployment,
	)
	// Output:
}
