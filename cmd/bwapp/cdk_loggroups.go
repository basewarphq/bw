package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type LogGroupsCmd struct {
	Deployment string `arg:"" required:"" help:"Deployment name (e.g., Staging, Prod)."`
}

func (c *LogGroupsCmd) Run(cfg *projcfg.Config) error {
	cdkDir := cfg.CdkDir()
	ctx := context.Background()

	out, err := cmdexec.Output(ctx, cdkDir, "cdk", "list")
	if err != nil {
		return err
	}

	for line := range strings.SplitSeq(out, "\n") {
		stack := strings.TrimSpace(line)
		if stack == "" || !strings.HasSuffix(stack, c.Deployment) {
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
			"--query", "Stacks[0].Outputs[?contains(OutputKey, 'LogGroup')]",
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
