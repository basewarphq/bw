package cfnvalidate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/cfnvalidate"
)

func TestPreBootstrapTemplate_Valid(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, `AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyPolicy:
    Type: AWS::IAM::ManagedPolicy
    Properties:
      PolicyDocument:
        Statement: []
Outputs:
  ExecutionPolicyArn:
    Value: !GetAtt MyPolicy.Arn
  PermissionBoundaryName:
    Value: my-boundary
`)
	if err := cfnvalidate.PreBootstrapTemplate(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreBootstrapTemplate_NoOutputs(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, `AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyPolicy:
    Type: AWS::IAM::ManagedPolicy
`)
	if err := cfnvalidate.PreBootstrapTemplate(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreBootstrapTemplate_NoResources(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, `AWSTemplateFormatVersion: "2010-09-09"
Outputs:
  Foo:
    Value: bar
`)
	err := cfnvalidate.PreBootstrapTemplate(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no Resources") {
		t.Errorf("error should mention missing Resources, got: %v", err)
	}
}

func TestPreBootstrapTemplate_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, `{{{invalid`)
	err := cfnvalidate.PreBootstrapTemplate(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPreBootstrapTemplate_FileNotFound(t *testing.T) {
	t.Parallel()
	err := cfnvalidate.PreBootstrapTemplate("/nonexistent/template.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
