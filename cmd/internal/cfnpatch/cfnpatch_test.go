package cfnpatch_test

import (
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/cfnpatch"
	"gopkg.in/yaml.v3"
)

const templateWithRules = `AWSTemplateFormatVersion: "2010-09-09"
Resources:
  StagingBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub "${AWS::StackName}-staging"
      LifecycleConfiguration:
        Rules:
          - Id: ExistingRule
            Status: Enabled
            Prefix: old/
            Expiration:
              Days: 30
`

const templateNoStagingBucket = `AWSTemplateFormatVersion: "2010-09-09"
Resources:
  OtherBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: other
`

func TestAddDevSlotLifecycle(t *testing.T) {
	t.Parallel()
	out, err := cfnpatch.AddDevSlotLifecycle([]byte(templateWithRules), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(out)

	if !strings.Contains(result, "CleanupDevSlotClaims") {
		t.Error("output should contain CleanupDevSlotClaims rule")
	}
	if !strings.Contains(result, "dev-slots/") {
		t.Error("output should contain dev-slots/ prefix")
	}
	if !strings.Contains(result, "ExistingRule") {
		t.Error("existing rule should be preserved")
	}
	if !strings.Contains(result, "!Sub") {
		t.Error("CloudFormation !Sub tag should be preserved")
	}
}

func TestAddDevSlotLifecycle_Idempotent(t *testing.T) {
	t.Parallel()
	out1, err := cfnpatch.AddDevSlotLifecycle([]byte(templateWithRules), 7)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	out2, err := cfnpatch.AddDevSlotLifecycle(out1, 14)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	result := string(out2)
	count := strings.Count(result, "CleanupDevSlotClaims")
	if count != 1 {
		t.Errorf("expected exactly 1 CleanupDevSlotClaims rule, got %d", count)
	}

	if !strings.Contains(result, "14") {
		t.Error("expiration days should be updated to 14")
	}
}

func TestAddDevSlotLifecycle_NoStagingBucket(t *testing.T) {
	t.Parallel()
	_, err := cfnpatch.AddDevSlotLifecycle([]byte(templateNoStagingBucket), 7)
	if err == nil {
		t.Fatal("expected error for template without StagingBucket")
	}
	if !strings.Contains(err.Error(), "StagingBucket") {
		t.Errorf("error should mention StagingBucket, got: %v", err)
	}
}

func TestAddDevSlotLifecycle_RealTemplate(t *testing.T) {
	t.Parallel()

	original := []byte(bootstrapTemplateSnapshot)

	patched, err := cfnpatch.AddDevSlotLifecycle(original, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var origDoc, patchedDoc yaml.Node
	if err := yaml.Unmarshal(original, &origDoc); err != nil {
		t.Fatalf("parsing original: %v", err)
	}
	if err := yaml.Unmarshal(patched, &patchedDoc); err != nil {
		t.Fatalf("parsing patched: %v", err)
	}

	origRoot := origDoc.Content[0]
	patchedRoot := patchedDoc.Content[0]

	for i := 0; i < len(origRoot.Content)-1; i += 2 {
		key := origRoot.Content[i].Value
		if key == "Resources" {
			continue
		}
		origYAML, _ := yaml.Marshal(origRoot.Content[i+1])
		patchedVal := findKey(t, patchedRoot, key)
		patchedYAML, _ := yaml.Marshal(patchedVal)
		if string(origYAML) != string(patchedYAML) {
			t.Errorf("top-level key %q was modified by patching", key)
		}
	}

	origResources := findKey(t, origRoot, "Resources")
	patchedResources := findKey(t, patchedRoot, "Resources")

	for i := 0; i < len(origResources.Content)-1; i += 2 {
		key := origResources.Content[i].Value
		if key == "StagingBucket" {
			continue
		}
		origYAML, _ := yaml.Marshal(origResources.Content[i+1])
		patchedVal := findKey(t, patchedResources, key)
		patchedYAML, _ := yaml.Marshal(patchedVal)
		if string(origYAML) != string(patchedYAML) {
			t.Errorf("resource %q was modified by patching", key)
		}
	}

	origBucket := findKey(t, origResources, "StagingBucket")
	patchedBucket := findKey(t, patchedResources, "StagingBucket")

	for i := 0; i < len(origBucket.Content)-1; i += 2 {
		key := origBucket.Content[i].Value
		if key == "Properties" {
			continue
		}
		origYAML, _ := yaml.Marshal(origBucket.Content[i+1])
		patchedVal := findKey(t, patchedBucket, key)
		patchedYAML, _ := yaml.Marshal(patchedVal)
		if string(origYAML) != string(patchedYAML) {
			t.Errorf("StagingBucket.%s was modified by patching", key)
		}
	}

	origProps := findKey(t, origBucket, "Properties")
	patchedProps := findKey(t, patchedBucket, "Properties")

	for i := 0; i < len(origProps.Content)-1; i += 2 {
		key := origProps.Content[i].Value
		if key == "LifecycleConfiguration" {
			continue
		}
		origYAML, _ := yaml.Marshal(origProps.Content[i+1])
		patchedVal := findKey(t, patchedProps, key)
		patchedYAML, _ := yaml.Marshal(patchedVal)
		if string(origYAML) != string(patchedYAML) {
			t.Errorf("StagingBucket.Properties.%s was modified by patching", key)
		}
	}

	patchedLifecycle := findKey(t, patchedProps, "LifecycleConfiguration")
	patchedRules := findKey(t, patchedLifecycle, "Rules")

	origLifecycle := findKey(t, origProps, "LifecycleConfiguration")
	origRules := findKey(t, origLifecycle, "Rules")

	if len(patchedRules.Content) != len(origRules.Content)+1 {
		t.Fatalf("expected %d rules, got %d", len(origRules.Content)+1, len(patchedRules.Content))
	}

	for i, origRule := range origRules.Content {
		origYAML, _ := yaml.Marshal(origRule)
		patchedYAML, _ := yaml.Marshal(patchedRules.Content[i])
		if string(origYAML) != string(patchedYAML) {
			t.Errorf("existing lifecycle rule %d was modified", i)
		}
	}

	addedRule := patchedRules.Content[len(patchedRules.Content)-1]
	addedYAML, _ := yaml.Marshal(addedRule)
	added := string(addedYAML)
	if !strings.Contains(added, "CleanupDevSlotClaims") {
		t.Error("added rule should have Id CleanupDevSlotClaims")
	}
	if !strings.Contains(added, "dev-slots/") {
		t.Error("added rule should have Prefix dev-slots/")
	}
	if !strings.Contains(added, "ExpirationInDays") {
		t.Error("added rule should have ExpirationInDays")
	}
}

func findKey(t *testing.T, node *yaml.Node, key string) *yaml.Node {
	t.Helper()
	if node.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping node when looking for key %q", key)
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	t.Fatalf("key %q not found", key)
	return nil
}
