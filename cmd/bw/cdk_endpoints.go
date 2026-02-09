package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/basewarphq/bw/bwcdk/bwcdkutil"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type EndpointsCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Staging, Prod). Defaults to claimed dev slot."`
}

func (c *EndpointsCmd) Run(cfg *projcfg.Config) error {
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

	for line := range strings.SplitSeq(out, "\n") {
		stack := strings.TrimSpace(line)
		if stack == "" || !strings.HasSuffix(stack, deployment) {
			continue
		}

		ident := bwcdkutil.ExtractRegionIdent(stack)
		if ident == "" {
			continue
		}
		region, ok := bwcdkutil.RegionForIdent(ident)
		if !ok {
			continue
		}

		fmt.Fprintf(os.Stdout, "=== %s (%s) ===\n", stack, region)

		out, err := cmdexec.Output(ctx, cdkDir, "aws", "cloudformation", "describe-stacks",
			"--no-cli-pager",
			"--region", region,
			"--stack-name", stack,
			"--query", "Stacks[0].Outputs[?contains(OutputKey, 'GatewayURL')]",
			"--output", "table",
		)
		if err != nil {
			fmt.Fprintln(os.Stdout, "(not deployed)")
		} else {
			fmt.Fprint(os.Stdout, out)
		}
		fmt.Fprintln(os.Stdout)
	}

	return nil
}
