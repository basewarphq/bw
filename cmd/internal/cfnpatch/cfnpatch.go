package cfnpatch

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

const ruleID = "CleanupDevSlotClaims"

func AddDevSlotLifecycle(templateYAML []byte, expirationDays int) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(templateYAML, &doc); err != nil {
		return nil, errors.Wrap(err, "parsing template YAML")
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, errors.New("invalid YAML document")
	}

	resources, err := mappingValue(doc.Content[0], "Resources")
	if err != nil {
		return nil, err
	}

	bucket, err := mappingValue(resources, "StagingBucket")
	if err != nil {
		return nil, errors.Wrap(err, "in Resources")
	}

	props, err := mappingValue(bucket, "Properties")
	if err != nil {
		return nil, errors.Wrap(err, "in StagingBucket")
	}

	lifecycleCfg, err := mappingValue(props, "LifecycleConfiguration")
	if err != nil {
		return nil, errors.Wrap(err, "in StagingBucket.Properties")
	}

	rules, err := mappingValue(lifecycleCfg, "Rules")
	if err != nil {
		return nil, errors.Wrap(err, "in StagingBucket.Properties.LifecycleConfiguration")
	}

	if rules.Kind != yaml.SequenceNode {
		return nil, errors.New("LifecycleConfiguration.Rules is not a sequence")
	}

	newRule := buildRuleNode(expirationDays)

	if idx := findRuleByID(rules, ruleID); idx >= 0 {
		rules.Content[idx] = newRule
	} else {
		rules.Content = append(rules.Content, newRule)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling patched template")
	}
	return out, nil
}

func mappingValue(node *yaml.Node, key string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, errors.Newf("expected mapping node for key %q", key)
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1], nil
		}
	}
	return nil, errors.Newf("key %q not found", key)
}

func findRuleByID(rules *yaml.Node, id string) int {
	for i, rule := range rules.Content {
		if rule.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j < len(rule.Content)-1; j += 2 {
			if rule.Content[j].Value == "Id" && rule.Content[j+1].Value == id {
				return i
			}
		}
	}
	return -1
}

func buildRuleNode(expirationDays int) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "Id"},
			{Kind: yaml.ScalarNode, Value: ruleID},
			{Kind: yaml.ScalarNode, Value: "Status"},
			{Kind: yaml.ScalarNode, Value: "Enabled"},
			{Kind: yaml.ScalarNode, Value: "Prefix"},
			{Kind: yaml.ScalarNode, Value: "dev-slots/"},
			{Kind: yaml.ScalarNode, Value: "ExpirationInDays"},
			{Kind: yaml.ScalarNode, Value: strconv.Itoa(expirationDays), Tag: "!!int"},
		},
	}
}
