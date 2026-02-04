//nolint:paralleltest // jsii runtime doesn't support parallel tests
package bwcdk1psync_test

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdk1psync"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
)

// testConfig returns a Config for testing.
func testConfig() *bwcdkutil.Config {
	return &bwcdkutil.Config{
		Qualifier:        "testqual",
		PrimaryRegion:    "us-east-1",
		SecondaryRegions: []string{"eu-west-1"},
		Deployments:      []string{"Stag", "Prod"},
		BaseDomainName:   "example.com",
	}
}

// validSAMLMetadata returns a minimal valid SAML metadata XML document.
func validSAMLMetadata() *string {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://1password.com">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://example.com/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>`
	return jsii.String(xml)
}

func TestNewProvider_PrimaryRegion(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})

	// Should not panic
	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})
}

func TestNewProvider_SecondaryRegion(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("eu-west-1"),
			Account: jsii.String("123456789012"),
		},
	})

	// Should not panic - secondary region looks up from primary
	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})
}

func TestNewProvider_EmptyMetadata_Panics(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty SAML metadata")
		}
	}()

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: jsii.String(""),
	})
}

func TestNewProvider_PlaceholderMetadata_Panics(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for placeholder SAML metadata")
		}
	}()

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: jsii.String("<!-- TODO: Download from 1Password -->"),
	})
}

func TestNewSyncRole_PrimaryRegion(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	// First create the provider so the SSM parameter exists
	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	syncRole := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String("IH75D4N7CP6JCAEATQMBNETCHQ"),
	})

	if syncRole == nil {
		t.Error("NewSyncRole should not return nil")
	}
	if syncRole.SecretRef() == nil {
		t.Error("SecretRef() should not return nil")
	}
}

func TestNewSyncRole_SecondaryRegion(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("eu-west-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	// Secondary region doesn't create the role but still returns a valid SyncRole
	syncRole := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String("IH75D4N7CP6JCAEATQMBNETCHQ"),
	})

	if syncRole == nil {
		t.Error("NewSyncRole should not return nil in secondary region")
	}
	if syncRole.SecretRef() == nil {
		t.Error("SecretRef() should not return nil in secondary region")
	}
}

func TestNewSyncRole_InvalidSubject_Panics(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for TODO placeholder SAML subject")
		}
	}()

	bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String("TODO: get from 1Password"),
	})
}

func TestNewSyncRole_EmptySubject_Panics(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty SAML subject")
		}
	}()

	bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String(""),
	})
}

func TestSecretRef_SecretName(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	syncRole := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String("IH75D4N7CP6JCAEATQMBNETCHQ"),
	})

	secretName := syncRole.SecretRef().SecretName()
	if secretName == nil {
		t.Fatal("SecretName() should not return nil")
	}

	// Format: {qualifier}/{deployment}/{identifier} (all lowercase)
	expected := "testqual/stag/main"
	if *secretName != expected {
		t.Errorf("SecretName() = %q, want %q", *secretName, expected)
	}
}

func TestSecretRef_SecretName_NoIdentifier(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Prod")

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	syncRole := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		SAMLSubject: jsii.String("IH75D4N7CP6JCAEATQMBNETCHQ"),
	})

	secretName := syncRole.SecretRef().SecretName()
	if secretName == nil {
		t.Fatal("SecretName() should not return nil")
	}

	// Format: {qualifier}/{deployment}/ (empty identifier)
	expected := "testqual/prod/"
	if *secretName != expected {
		t.Errorf("SecretName() = %q, want %q", *secretName, expected)
	}
}

func TestSecretRef_GrantRead(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	syncRole := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String("IH75D4N7CP6JCAEATQMBNETCHQ"),
	})

	// Create a Lambda function to grant permissions to
	fn := awslambda.NewFunction(stack, jsii.String("TestFn"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_NODEJS_22_X(),
		Handler: jsii.String("index.handler"),
		Code:    awslambda.Code_FromInline(jsii.String("exports.handler = async () => {}")),
	})

	// Should not panic
	syncRole.SecretRef().GrantRead(fn)
}

func TestMultipleSyncRoles_SameStack(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	bwcdkutil.StoreConfig(app, testConfig())
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region:  jsii.String("us-east-1"),
			Account: jsii.String("123456789012"),
		},
	})
	bwcdkutil.StoreDeploymentIdent(stack, "Stag")

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: validSAMLMetadata(),
	})

	syncRole1 := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Main"),
		SAMLSubject: jsii.String("IH75D4N7CP6JCAEATQMBNETCHQ"),
	})
	syncRole2 := bwcdk1psync.NewSyncRole(stack, bwcdk1psync.SyncRoleProps{
		Identifier:  jsii.String("Backend"),
		SAMLSubject: jsii.String("XY85E5O8DP7KDBEBURONCOFDIQ"),
	})

	name1 := syncRole1.SecretRef().SecretName()
	name2 := syncRole2.SecretRef().SecretName()

	if *name1 == *name2 {
		t.Errorf("different identifiers should produce different secret names: %q vs %q", *name1, *name2)
	}
}

func TestRoleARNOutputKey(t *testing.T) {
	if got := bwcdk1psync.RoleARNOutputKey(""); got != "OnePasswordSyncRoleARN" {
		t.Errorf("RoleARNOutputKey('') = %q, want 'OnePasswordSyncRoleARN'", got)
	}
	if got := bwcdk1psync.RoleARNOutputKey("Main"); got != "OnePasswordSyncRoleARNMain" {
		t.Errorf("RoleARNOutputKey('Main') = %q, want 'OnePasswordSyncRoleARNMain'", got)
	}
}

func TestSecretNameOutputKey(t *testing.T) {
	if got := bwcdk1psync.SecretNameOutputKey(""); got != "OnePasswordSyncSecretName" {
		t.Errorf("SecretNameOutputKey('') = %q, want 'OnePasswordSyncSecretName'", got)
	}
	if got := bwcdk1psync.SecretNameOutputKey("StagMain"); got != "OnePasswordSyncSecretNameStagMain" {
		t.Errorf("SecretNameOutputKey('StagMain') = %q, want 'OnePasswordSyncSecretNameStagMain'", got)
	}
}
