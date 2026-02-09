package bwcdkutil

// legacy.go provides support for legacy projects that use custom, project-specific
// region identifiers instead of the standard 4-character identifiers defined in
// RegionIdents (e.g., "De" for eu-central-1 instead of "Euc1").
//
// Without this, adopting bwcdkutil in a legacy project would change CloudFormation
// stack names (e.g., "bcxDeShared" → "bcxEuc1Shared"), causing CloudFormation to
// replace or destroy existing infrastructure.
//
// Usage: set LegacyRegionIdent: true in AppConfig, and provide context keys
// "{prefix}region-ident-{region}" (e.g., "bw-region-ident-eu-central-1": "De")
// in cdk.context.json.

import (
	"fmt"

	"github.com/aws/constructs-go/constructs/v10"
)

// readLegacyRegionIdents reads custom region identifiers from CDK context.
// For each region in regions, it looks up the key "{prefix}region-ident-{region}".
// Returns a map[string]string of region → custom ident, and any errors.
func readLegacyRegionIdents(
	scope constructs.Construct, prefix string, regions []string, errs []string,
) (map[string]string, []string) {
	idents := make(map[string]string, len(regions))
	for _, region := range regions {
		key := fmt.Sprintf("%sregion-ident-%s", prefix, region)
		val, newErrs := readContextString(scope, key, errs)
		errs = newErrs
		if val != "" {
			idents[region] = val
		}
	}
	return idents, errs
}
