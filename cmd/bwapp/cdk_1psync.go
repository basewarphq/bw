package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
	"github.com/basewarphq/bwapp/cmd/internal/cfnread"
	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
	"github.com/cockroachdb/errors"
)

type OnePasswordSyncCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
}

func (c *OnePasswordSyncCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	deployment := c.Deployment
	if deployment == "" {
		claim, err := ensureClaim(ctx, cfg)
		if err != nil {
			return err
		}
		deployment = claim.Slot
	}

	cdkDir := cfg.CdkDir()

	out, err := cmdexec.Output(ctx, cdkDir, "cdk", "list")
	if err != nil {
		return err
	}

	var sharedStack, deploymentStack, primaryRegion string
	for line := range strings.SplitSeq(out, "\n") {
		stack := strings.TrimSpace(line)
		if stack == "" {
			continue
		}

		if sharedStack == "" && strings.HasSuffix(stack, "Shared") {
			sharedStack = stack
		}
		if deploymentStack == "" && strings.HasSuffix(stack, deployment) {
			deploymentStack = stack
		}
	}

	if sharedStack == "" {
		return errors.New("no shared stack found")
	}
	if deploymentStack == "" {
		return errors.Newf("no deployment stack found for %s", deployment)
	}

	ident := bwcdkutil.ExtractRegionIdent(sharedStack)
	if ident == "" {
		return errors.Newf("cannot extract region from stack %s", sharedStack)
	}
	region, ok := bwcdkutil.RegionForIdent(ident)
	if !ok {
		return errors.Newf("unknown region identifier %s", ident)
	}
	primaryRegion = region

	sharedOutputs, err := cfnread.StackOutputs(ctx, primaryRegion, sharedStack)
	if err != nil {
		return err
	}

	deployOutputs, err := cfnread.StackOutputs(ctx, primaryRegion, deploymentStack)
	if err != nil {
		return err
	}

	samlARN := sharedOutputs["OnePasswordSAMLProviderARN"]
	roleARN := outputContaining(deployOutputs, "OnePasswordSyncRoleARN")
	secretName := outputContaining(deployOutputs, "OnePasswordSyncSecretName")

	fmt.Fprintf(os.Stdout, "=== 1Password Sync Configuration for %s ===\n", deployment)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Copy these values into 1Password:")
	fmt.Fprintln(os.Stdout, "  Developer > View Environments > [env] > Destinations > Configure AWS")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "SAML provider ARN:")
	fmt.Fprintf(os.Stdout, "  %s\n", valueOrNotFound(samlARN, "deploy shared stack first"))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "IAM role ARN:")
	fmt.Fprintf(os.Stdout, "  %s\n", valueOrNotFound(roleARN, "deploy deployment stack first"))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Target region:")
	fmt.Fprintf(os.Stdout, "  %s\n", primaryRegion)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Target secret name:")
	fmt.Fprintf(os.Stdout, "  %s\n", valueOrNotFound(secretName, "deploy deployment stack first"))

	return nil
}

func outputContaining(outputs map[string]string, substr string) string {
	for k, v := range outputs {
		if strings.Contains(k, substr) {
			return v
		}
	}
	return ""
}

func valueOrNotFound(val, hint string) string {
	if val == "" {
		return "(not found - " + hint + ")"
	}
	return val
}
