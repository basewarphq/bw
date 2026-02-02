//nolint:paralleltest // jsii runtime doesn't support parallel tests
package bwcdklwalambda_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdklwalambda"
)

// testEntry is a valid entry path pointing to an actual Go command in the repo.
// Tests requiring CDK runtime must run from the module root.
var testEntry = "backend/cmd/coreback"

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

func TestParseEntry(t *testing.T) {
	tests := []struct {
		name          string
		entry         string
		wantComponent string
		wantCommand   string
		wantErr       bool
	}{
		{
			name:          "valid simple path",
			entry:         "backend/cmd/coreback",
			wantComponent: "backend",
			wantCommand:   "coreback",
		},
		{
			name:          "valid deep path",
			entry:         "some/deep/path/component/cmd/handler",
			wantComponent: "component",
			wantCommand:   "handler",
		},
		{
			name:          "valid with trailing slash normalized",
			entry:         "backend/cmd/api",
			wantComponent: "backend",
			wantCommand:   "api",
		},
		{
			name:    "missing cmd segment",
			entry:   "backend/coreback",
			wantErr: true,
		},
		{
			name:    "empty entry",
			entry:   "",
			wantErr: true,
		},
		{
			name:    "only cmd",
			entry:   "cmd/handler",
			wantErr: true,
		},
		{
			name:    "empty command after cmd",
			entry:   "backend/cmd/",
			wantErr: true,
		},
		{
			name:    "empty component before cmd",
			entry:   "/cmd/handler",
			wantErr: true,
		},
		{
			name:    "cmd at wrong position",
			entry:   "cmd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, command, err := bwcdklwalambda.ParseEntry(tt.entry)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if component != tt.wantComponent {
				t.Errorf("component = %q, want %q", component, tt.wantComponent)
			}
			if command != tt.wantCommand {
				t.Errorf("command = %q, want %q", command, tt.wantCommand)
			}
		})
	}
}

func TestNew_WithoutInvokePath(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})

	lambda := bwcdklwalambda.New(stack, bwcdklwalambda.Props{
		Entry: jsii.String(testEntry),
	})

	if lambda.Name() != "BackendCoreback" {
		t.Errorf("Name() = %q, want %q", lambda.Name(), "BackendCoreback")
	}
	if lambda.Function() == nil {
		t.Error("Function() should not be nil")
	}
	if lambda.LogGroup() == nil {
		t.Error("LogGroup() should not be nil")
	}
}

func TestNew_WithInvokePath(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})

	lambda := bwcdklwalambda.New(stack, bwcdklwalambda.Props{
		Entry:      jsii.String(testEntry),
		InvokePath: jsii.String("/l/authorize"),
	})

	if lambda.Name() != "BackendCorebackAuthorize" {
		t.Errorf("Name() = %q, want %q", lambda.Name(), "BackendCorebackAuthorize")
	}
	if lambda.Function() == nil {
		t.Error("Function() should not be nil")
	}
	if lambda.LogGroup() == nil {
		t.Error("LogGroup() should not be nil")
	}
}

func TestNew_WithInvokePathKebabCase(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})

	lambda := bwcdklwalambda.New(stack, bwcdklwalambda.Props{
		Entry:      jsii.String(testEntry),
		InvokePath: jsii.String("/l/some-handler"),
	})

	if lambda.Name() != "BackendCorebackSomeHandler" {
		t.Errorf("Name() = %q, want %q", lambda.Name(), "BackendCorebackSomeHandler")
	}
}

func TestNew_InvalidEntry(t *testing.T) {
	defer jsii.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid entry")
		}
	}()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})

	bwcdklwalambda.New(stack, bwcdklwalambda.Props{
		Entry: jsii.String("invalid/path"),
	})
}

func TestNew_InvalidInvokePath(t *testing.T) {
	defer jsii.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid invoke path")
		}
	}()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), &awscdk.StackProps{
		Env: &awscdk.Environment{
			Region: jsii.String("us-east-1"),
		},
	})

	bwcdklwalambda.New(stack, bwcdklwalambda.Props{
		Entry:      jsii.String(testEntry),
		InvokePath: jsii.String("/authorize"), // missing /l/ prefix
	})
}
