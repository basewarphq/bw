package cdkctx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/basewarphq/bw/bwcdk/bwcdkutil"
	"github.com/cockroachdb/errors"
)

type CDKContext struct {
	Qualifier     string
	Prefix        string
	PrimaryRegion string
	Deployments   []string
	RegionIdents  map[string]string
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

	regionIdents := make(map[string]string)
	regionIdentPrefix := prefix + "region-ident-"
	for key := range ctxMap {
		if !strings.HasPrefix(key, regionIdentPrefix) {
			continue
		}
		region := strings.TrimPrefix(key, regionIdentPrefix)
		ident, err := getString(ctxMap, key)
		if err != nil {
			return nil, errors.Wrapf(err, "in %s", ctxFile)
		}
		regionIdents[ident] = region
	}
	for region, ident := range bwcdkutil.RegionIdents {
		if _, ok := regionIdents[ident]; !ok {
			regionIdents[ident] = region
		}
	}

	return &CDKContext{
		Qualifier:     qualifier,
		Prefix:        prefix,
		PrimaryRegion: primaryRegion,
		Deployments:   deployments,
		RegionIdents:  regionIdents,
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

func (c *CDKContext) ResolveStackRegion(stackName string) (string, bool) {
	rest := strings.TrimPrefix(stackName, c.Qualifier)
	if rest == stackName {
		return "", false
	}

	idents := make([]string, 0, len(c.RegionIdents))
	for ident := range c.RegionIdents {
		idents = append(idents, ident)
	}
	sort.Slice(idents, func(i, j int) bool {
		return len(idents[i]) > len(idents[j])
	})

	for _, ident := range idents {
		if strings.HasPrefix(rest, ident) {
			return c.RegionIdents[ident], true
		}
	}
	return "", false
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
