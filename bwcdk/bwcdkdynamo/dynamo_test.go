package bwcdkdynamo_test

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bw/bwcdk/bwcdkdynamo"
	"github.com/basewarphq/bw/bwcdk/bwcdkutil"
)

// testConfig returns a Config for testing.
func testConfig() *bwcdkutil.Config {
	return &bwcdkutil.Config{
		Qualifier:        "testqual",
		PrimaryRegion:    "us-east-1",
		SecondaryRegions: []string{"eu-west-1"},
		Deployments:      []string{"dev", "Prod"},
		BaseDomainName:   "example.com",
	}
}

func TestNew_PrimaryRegion_DefaultIdentifier(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	dynamo := bwcdkdynamo.New(stack, bwcdkdynamo.Props{})

	if dynamo.Table() == nil {
		t.Error("Table() should not be nil")
	}
}

func TestNew_PrimaryRegion_CustomIdentifier(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	dynamo := bwcdkdynamo.New(stack, bwcdkdynamo.Props{
		Identifier: jsii.String("users"),
	})

	if dynamo.Table() == nil {
		t.Error("Table() should not be nil")
	}
}

func TestNew_MultipleTablesInSameStack(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	dynamo1 := bwcdkdynamo.New(stack, bwcdkdynamo.Props{
		Identifier: jsii.String("main"),
	})
	dynamo2 := bwcdkdynamo.New(stack, bwcdkdynamo.Props{
		Identifier: jsii.String("events"),
	})

	if dynamo1.Table() == nil {
		t.Error("first Table() should not be nil")
	}
	if dynamo2.Table() == nil {
		t.Error("second Table() should not be nil")
	}
	if *dynamo1.Table().TableName() == *dynamo2.Table().TableName() {
		t.Error("tables should have different names")
	}
}

func TestNew_GrantReadData(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	dynamo := bwcdkdynamo.New(stack, bwcdkdynamo.Props{})

	// Create a Lambda function to grant permissions to
	fn := awslambda.NewFunction(stack, jsii.String("TestFn"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_NODEJS_22_X(),
		Handler: jsii.String("index.handler"),
		Code:    awslambda.Code_FromInline(jsii.String("exports.handler = async () => {}")),
	})

	// Should not panic
	dynamo.GrantReadData(fn)
}

func TestNew_GrantReadWriteData(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	dynamo := bwcdkdynamo.New(stack, bwcdkdynamo.Props{})

	// Create a Lambda function to grant permissions to
	fn := awslambda.NewFunction(stack, jsii.String("TestFn"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_NODEJS_22_X(),
		Handler: jsii.String("index.handler"),
		Code:    awslambda.Code_FromInline(jsii.String("exports.handler = async () => {}")),
	})

	// Should not panic
	dynamo.GrantReadWriteData(fn)
}

func TestNew_ProdDeployment(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Prod")

	dynamo := bwcdkdynamo.New(stack, bwcdkdynamo.Props{})

	if dynamo.Table() == nil {
		t.Error("Table() should not be nil")
	}
}
