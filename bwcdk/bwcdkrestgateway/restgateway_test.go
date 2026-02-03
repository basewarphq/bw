//nolint:paralleltest // jsii runtime doesn't support parallel tests
package bwcdkrestgateway_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkrestgateway"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
)

// testEntry is a valid entry path pointing to an actual Go command in the repo.
// Tests requiring CDK runtime must run from the module root.
var testEntry = "backend/cmd/coreback"

// testConfig returns a Config for testing.
func testConfig() *bwcdkutil.Config {
	return &bwcdkutil.Config{
		Qualifier:      "testqual",
		PrimaryRegion:  "us-east-1",
		Deployments:    []string{"dev", "Prod"},
		BaseDomainName: "example.com",
	}
}

func init() {
	// Change to module root so CDK can find the entry path.
	// Find go.mod to locate module root.
	dir, _ := os.Getwd()
	for dir != "/" {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			_ = os.Chdir(dir)
			break
		}
		dir = filepath.Dir(dir)
	}
}

func TestNew_WithoutAuthorizer(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("eu-west-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	hostedZone := awsroute53.NewHostedZone(stack, jsii.String("Zone"), &awsroute53.HostedZoneProps{
		ZoneName: jsii.String("example.com"),
	})

	certificate := awscertificatemanager.NewCertificate(stack, jsii.String("Cert"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String("*.example.com"),
	})

	gateway := bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry:        jsii.String(testEntry),
		PublicRoutes: &[]*string{jsii.String("/g/{proxy+}")},
		HostedZone:   hostedZone,
		Certificate:  certificate,
		Subdomain:    jsii.String("api"),
	})

	if gateway.Lambda() == nil {
		t.Error("Lambda() should not be nil")
	}
	if gateway.AuthorizerLambda() != nil {
		t.Error("AuthorizerLambda() should be nil when no authorizer configured")
	}
	if gateway.RestApi() == nil {
		t.Error("RestApi() should not be nil")
	}
	if gateway.AccessLogGroup() == nil {
		t.Error("AccessLogGroup() should not be nil")
	}

	wantDomainName := "dev-euw1-api.example.com"
	if gateway.DomainName() != wantDomainName {
		t.Errorf("DomainName() = %q, want %q", gateway.DomainName(), wantDomainName)
	}

	wantGlobalDomainName := "dev-api.example.com"
	if gateway.GlobalDomainName() != wantGlobalDomainName {
		t.Errorf("GlobalDomainName() = %q, want %q", gateway.GlobalDomainName(), wantGlobalDomainName)
	}
}

func TestNew_WithAuthorizer(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Prod")

	hostedZone := awsroute53.NewHostedZone(stack, jsii.String("Zone"), &awsroute53.HostedZoneProps{
		ZoneName: jsii.String("basewarp.app"),
	})

	certificate := awscertificatemanager.NewCertificate(stack, jsii.String("Cert"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String("*.basewarp.app"),
	})

	gateway := bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry:        jsii.String(testEntry),
		PublicRoutes: &[]*string{jsii.String("/g/{proxy+}")},
		HostedZone:   hostedZone,
		Certificate:  certificate,
		Subdomain:    jsii.String("api"),
		Authorizer:   &bwcdkrestgateway.AuthorizerProps{},
	})

	if gateway.Lambda() == nil {
		t.Error("Lambda() should not be nil")
	}
	if gateway.AuthorizerLambda() == nil {
		t.Error("AuthorizerLambda() should not be nil when authorizer is configured")
	}
	if gateway.RestApi() == nil {
		t.Error("RestApi() should not be nil")
	}

	// Prod deployment omits deployment prefix from domain names.
	wantDomainName := "use1-api.basewarp.app"
	if gateway.DomainName() != wantDomainName {
		t.Errorf("DomainName() = %q, want %q", gateway.DomainName(), wantDomainName)
	}

	wantGlobalDomainName := "api.basewarp.app"
	if gateway.GlobalDomainName() != wantGlobalDomainName {
		t.Errorf("GlobalDomainName() = %q, want %q", gateway.GlobalDomainName(), wantGlobalDomainName)
	}

	// Verify authorizer Lambda has different name (includes PassThroughPath suffix)
	if gateway.Lambda().Name() == gateway.AuthorizerLambda().Name() {
		t.Errorf("Lambda and AuthorizerLambda should have different names, both are %q", gateway.Lambda().Name())
	}
	if gateway.AuthorizerLambda().Name() != "BackendCorebackAuthorize" {
		t.Errorf("AuthorizerLambda().Name() = %q, want %q", gateway.AuthorizerLambda().Name(), "BackendCorebackAuthorize")
	}
}

func TestNew_MultipleRoutes(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("eu-west-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	hostedZone := awsroute53.NewHostedZone(stack, jsii.String("Zone"), &awsroute53.HostedZoneProps{
		ZoneName: jsii.String("example.com"),
	})

	certificate := awscertificatemanager.NewCertificate(stack, jsii.String("Cert"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String("*.example.com"),
	})

	gateway := bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry: jsii.String(testEntry),
		PublicRoutes: &[]*string{
			jsii.String("/g/{proxy+}"),
			jsii.String("/health"),
			jsii.String("/api/v1/{proxy+}"),
		},
		HostedZone:  hostedZone,
		Certificate: certificate,
		Subdomain:   jsii.String("api"),
	})

	if gateway.RestApi() == nil {
		t.Error("RestApi() should not be nil")
	}
}

func TestNew_WithEnvironment(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("eu-west-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "dev")

	hostedZone := awsroute53.NewHostedZone(stack, jsii.String("Zone"), &awsroute53.HostedZoneProps{
		ZoneName: jsii.String("example.com"),
	})

	certificate := awscertificatemanager.NewCertificate(stack, jsii.String("Cert"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String("*.example.com"),
	})

	env := map[string]*string{
		"MY_VAR": jsii.String("my-value"),
	}

	gateway := bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry:        jsii.String(testEntry),
		PublicRoutes: &[]*string{jsii.String("/g/{proxy+}")},
		Environment:  &env,
		HostedZone:   hostedZone,
		Certificate:  certificate,
		Subdomain:    jsii.String("api"),
	})

	if gateway.Lambda() == nil {
		t.Error("Lambda() should not be nil")
	}
}

func TestNew_DifferentSubdomains(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("ap-southeast-1"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "staging")

	hostedZone := awsroute53.NewHostedZone(stack, jsii.String("Zone"), &awsroute53.HostedZoneProps{
		ZoneName: jsii.String("myapp.io"),
	})

	certificate := awscertificatemanager.NewCertificate(stack, jsii.String("Cert"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String("*.myapp.io"),
	})

	gateway := bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry:        jsii.String(testEntry),
		PublicRoutes: &[]*string{jsii.String("/webhook")},
		HostedZone:   hostedZone,
		Certificate:  certificate,
		Subdomain:    jsii.String("webhook"),
	})

	// ap-southeast-1 -> ase1
	wantDomainName := "staging-ase1-webhook.myapp.io"
	if gateway.DomainName() != wantDomainName {
		t.Errorf("DomainName() = %q, want %q", gateway.DomainName(), wantDomainName)
	}

	wantGlobalDomainName := "staging-webhook.myapp.io"
	if gateway.GlobalDomainName() != wantGlobalDomainName {
		t.Errorf("GlobalDomainName() = %q, want %q", gateway.GlobalDomainName(), wantGlobalDomainName)
	}
}
