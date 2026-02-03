//nolint:paralleltest // jsii runtime doesn't support parallel tests
package bwcdkutil_test

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
)

func TestResourceName_DeploymentStack(t *testing.T) {
	defer jsii.Close()

	tests := []struct {
		name   string
		label  string
		casing bwcdkutil.Casing
		want   string
	}{
		{
			name:   "camel case",
			label:  "ApiGateway",
			casing: bwcdkutil.CasingCamel,
			want:   "TestqualStagApiGateway",
		},
		{
			name:   "lower camel case",
			label:  "ApiGateway",
			casing: bwcdkutil.CasingLowerCamel,
			want:   "testqualStagApiGateway",
		},
		{
			name:   "snake case",
			label:  "ApiGateway",
			casing: bwcdkutil.CasingSnake,
			want:   "testqual_stag_api_gateway",
		},
		{
			name:   "screaming snake case",
			label:  "ApiGateway",
			casing: bwcdkutil.CasingScreamingSnake,
			want:   "TESTQUAL_STAG_API_GATEWAY",
		},
		{
			name:   "kebab case",
			label:  "ApiGateway",
			casing: bwcdkutil.CasingKebab,
			want:   "testqual-stag-api-gateway",
		},
		{
			name:   "screaming kebab case",
			label:  "ApiGateway",
			casing: bwcdkutil.CasingScreamingKebab,
			want:   "TESTQUAL-STAG-API-GATEWAY",
		},
		{
			name:   "kebab label converted to camel",
			label:  "my-lambda-function",
			casing: bwcdkutil.CasingCamel,
			want:   "TestqualStagMyLambdaFunction",
		},
		{
			name:   "snake label converted to kebab",
			label:  "my_lambda_function",
			casing: bwcdkutil.CasingKebab,
			want:   "testqual-stag-my-lambda-function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := awscdk.NewApp(nil)

			cfg := &bwcdkutil.Config{
				Qualifier:     "testqual",
				PrimaryRegion: "us-east-1",
				Deployments:   []string{"Stag", "Prod"},
				BaseDomainName: "example.com",
			}
			bwcdkutil.StoreConfig(app, cfg)

			stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
				Env: &awscdk.Environment{
					Region: jsii.String("us-east-1"),
				},
			})
			bwcdkutil.StoreDeploymentIdent(stack, "Stag")

			got := bwcdkutil.ResourceName(stack, tt.label, tt.casing)
			if got != tt.want {
				t.Errorf("ResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResourceName_SharedStack(t *testing.T) {
	defer jsii.Close()

	tests := []struct {
		name   string
		label  string
		casing bwcdkutil.Casing
		want   string
	}{
		{
			name:   "camel case without deployment",
			label:  "HostedZone",
			casing: bwcdkutil.CasingCamel,
			want:   "TestqualHostedZone",
		},
		{
			name:   "kebab case without deployment",
			label:  "HostedZone",
			casing: bwcdkutil.CasingKebab,
			want:   "testqual-hosted-zone",
		},
		{
			name:   "snake case without deployment",
			label:  "HostedZone",
			casing: bwcdkutil.CasingSnake,
			want:   "testqual_hosted_zone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := awscdk.NewApp(nil)

			cfg := &bwcdkutil.Config{
				Qualifier:      "testqual",
				PrimaryRegion:  "us-east-1",
				Deployments:    []string{"Prod"},
				BaseDomainName: "example.com",
			}
			bwcdkutil.StoreConfig(app, cfg)

			stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
				Env: &awscdk.Environment{
					Region: jsii.String("us-east-1"),
				},
			})

			got := bwcdkutil.ResourceName(stack, tt.label, tt.casing)
			if got != tt.want {
				t.Errorf("ResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}
