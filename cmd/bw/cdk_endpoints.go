package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type EndpointsCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Staging, Prod). Defaults to claimed dev slot."`
}

func (c *EndpointsCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	deployment, err := resolveDeployment(ctx, cfg, c.Deployment)
	if err != nil {
		return err
	}

	cdkDir := cfg.CdkDir()

	cctx, err := cdkctx.Load(cdkDir)
	if err != nil {
		return err
	}

	listArgs := append([]string{"list"}, cfg.Cdk.CdkArgs(cctx.Qualifier)...)
	out, err := cmdexec.Output(ctx, cdkDir, "cdk", listArgs...)
	if err != nil {
		return err
	}

	for line := range strings.SplitSeq(out, "\n") {
		stack := strings.TrimSpace(line)
		if stack == "" || !strings.HasSuffix(stack, deployment) {
			continue
		}

		region, ok := cctx.ResolveStackRegion(stack)
		if !ok {
			continue
		}

		fmt.Fprintf(os.Stdout, "=== %s (%s) ===\n", stack, region)

		awsArgs := make([]string, 0, 12+len(cfg.Cdk.AwsArgs()))
		awsArgs = append(awsArgs,
			"cloudformation", "describe-stacks",
			"--no-cli-pager",
			"--region", region,
			"--stack-name", stack,
			"--query", "Stacks[0].Outputs[?contains(OutputKey, 'GatewayURL')]",
			"--output", "table",
		)
		awsArgs = append(awsArgs, cfg.Cdk.AwsArgs()...)
		out, err := cmdexec.Output(ctx, cdkDir, "aws", awsArgs...)
		if err != nil {
			fmt.Fprintln(os.Stdout, "(not deployed)")
		} else {
			fmt.Fprint(os.Stdout, out)
		}
		fmt.Fprintln(os.Stdout)
	}

	return nil
}
