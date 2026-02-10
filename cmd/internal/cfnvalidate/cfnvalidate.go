package cfnvalidate

import (
	"os"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

func PreBootstrapTemplate(templatePath string) error {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return errors.Wrapf(err, "reading template %s", templatePath)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return errors.Wrap(err, "parsing template YAML")
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return errors.New("invalid YAML document")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return errors.New("template root is not a mapping")
	}

	if findMappingValue(root, "Resources") == nil {
		return errors.New("template has no Resources section")
	}

	return nil
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
