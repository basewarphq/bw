package cdkctx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cockroachdb/errors"
)

type CDKContext struct {
	Qualifier     string
	Prefix        string
	PrimaryRegion string
	Deployments   []string
}

func Load(cdkDir string) (*CDKContext, error) {
	qualifier, err := readQualifier(cdkDir)
	if err != nil {
		return nil, err
	}

	prefix := qualifier + "-"

	ctxFile := filepath.Join(cdkDir, "cdk.context.json")
	ctxData, err := os.ReadFile(ctxFile)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", ctxFile)
	}

	var ctxMap map[string]json.RawMessage
	if err := json.Unmarshal(ctxData, &ctxMap); err != nil {
		return nil, errors.Wrapf(err, "parsing %s", ctxFile)
	}

	primaryRegion, err := getString(ctxMap, prefix+"primary-region")
	if err != nil {
		return nil, errors.Wrapf(err, "in %s", ctxFile)
	}

	deployments, err := getStringSlice(ctxMap, prefix+"deployments")
	if err != nil {
		return nil, errors.Wrapf(err, "in %s", ctxFile)
	}

	return &CDKContext{
		Qualifier:     qualifier,
		Prefix:        prefix,
		PrimaryRegion: primaryRegion,
		Deployments:   deployments,
	}, nil
}

func (c *CDKContext) DevSlots() []string {
	var slots []string
	for _, d := range c.Deployments {
		if strings.HasPrefix(d, "Dev") {
			slots = append(slots, d)
		}
	}
	return slots
}

func (c *CDKContext) BootstrapBucket(accountID string) string {
	return "cdk-" + c.Qualifier + "-assets-" + accountID + "-" + c.PrimaryRegion
}

func (c *CDKContext) IsValidDeployment(name string) bool {
	return slices.Contains(c.Deployments, name)
}

func readQualifier(cdkDir string) (string, error) {
	cdkJSON := filepath.Join(cdkDir, "cdk.json")
	data, err := os.ReadFile(cdkJSON)
	if err != nil {
		return "", errors.Wrapf(err, "reading %s", cdkJSON)
	}

	var cfg struct {
		Context map[string]json.RawMessage `json:"context"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", errors.Wrapf(err, "parsing %s", cdkJSON)
	}

	raw, ok := cfg.Context["@aws-cdk/core:bootstrapQualifier"]
	if !ok {
		return "", errors.Newf("missing @aws-cdk/core:bootstrapQualifier in %s", cdkJSON)
	}

	var qualifier string
	if err := json.Unmarshal(raw, &qualifier); err != nil {
		return "", errors.Newf("@aws-cdk/core:bootstrapQualifier must be a string in %s", cdkJSON)
	}
	return qualifier, nil
}

func getString(m map[string]json.RawMessage, key string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", errors.Newf("context key %q is not set", key)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", errors.Newf("context key %q must be a string", key)
	}
	return s, nil
}

func getStringSlice(m map[string]json.RawMessage, key string) ([]string, error) {
	raw, ok := m[key]
	if !ok {
		return nil, errors.Newf("context key %q is not set", key)
	}
	var ss []string
	if err := json.Unmarshal(raw, &ss); err != nil {
		return nil, errors.Newf("context key %q must be an array of strings", key)
	}
	return ss, nil
}
