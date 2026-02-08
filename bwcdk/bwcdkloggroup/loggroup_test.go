package bwcdkloggroup_test

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkloggroup"
)

func TestNew_CreatesLogGroup(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), nil)

	lg := bwcdkloggroup.New(stack, "TestLogs", bwcdkloggroup.Props{
		Purpose: jsii.String("test logs"),
	})

	if lg.LogGroup() == nil {
		t.Error("LogGroup() should not be nil")
	}
}

func TestNew_CreatesOutput(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), nil)

	bwcdkloggroup.New(stack, "MyLogs", bwcdkloggroup.Props{
		Purpose: jsii.String("Lambda function logs"),
	})

	template := app.Synth(nil).GetStackByName(jsii.String("TestStack")).Template()

	templateJSON, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("failed to marshal template: %v", err)
	}

	var tmpl map[string]any
	if err := json.Unmarshal(templateJSON, &tmpl); err != nil {
		t.Fatalf("failed to unmarshal template: %v", err)
	}

	outputs, ok := tmpl["Outputs"].(map[string]any)
	if !ok {
		t.Fatal("template should have Outputs")
	}

	var foundOutput map[string]any
	for key, val := range outputs {
		if m, ok := val.(map[string]any); ok {
			if desc, ok := m["Description"].(string); ok && desc == "CloudWatch Log Group for Lambda function logs" {
				foundOutput = m
				break
			}
		}
		_ = key
	}
	if foundOutput == nil {
		t.Fatalf("template should have output with expected description, got outputs: %v", outputs)
	}
	output := foundOutput

	desc, ok := output["Description"].(string)
	if !ok || desc != "CloudWatch Log Group for Lambda function logs" {
		t.Errorf("Description = %q, want %q", desc, "CloudWatch Log Group for Lambda function logs")
	}
}

func TestNew_MultipleLogGroups(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("TestStack"), nil)

	lg1 := bwcdkloggroup.New(stack, "FirstLogs", bwcdkloggroup.Props{
		Purpose: jsii.String("first purpose"),
	})
	lg2 := bwcdkloggroup.New(stack, "SecondLogs", bwcdkloggroup.Props{
		Purpose: jsii.String("second purpose"),
	})

	if lg1.LogGroup() == nil {
		t.Error("first LogGroup() should not be nil")
	}
	if lg2.LogGroup() == nil {
		t.Error("second LogGroup() should not be nil")
	}

	template := app.Synth(nil).GetStackByName(jsii.String("TestStack")).Template()

	templateJSON, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("failed to marshal template: %v", err)
	}

	var tmpl map[string]any
	if err := json.Unmarshal(templateJSON, &tmpl); err != nil {
		t.Fatalf("failed to unmarshal template: %v", err)
	}

	outputs, ok := tmpl["Outputs"].(map[string]any)
	if !ok {
		t.Fatal("template should have Outputs")
	}

	foundFirst := false
	foundSecond := false
	for _, val := range outputs {
		desc := extractDescription(val)
		if desc == "CloudWatch Log Group for first purpose" {
			foundFirst = true
		}
		if desc == "CloudWatch Log Group for second purpose" {
			foundSecond = true
		}
	}
	if !foundFirst {
		t.Error("template should have output for first purpose")
	}
	if !foundSecond {
		t.Error("template should have output for second purpose")
	}
}

func extractDescription(val any) string {
	m, ok := val.(map[string]any)
	if !ok {
		return ""
	}
	desc, ok := m["Description"].(string)
	if !ok {
		return ""
	}
	return desc
}
