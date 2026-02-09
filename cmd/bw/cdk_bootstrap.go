package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type BootstrapCmd struct {
	ExecutionPolicies   string `name:"execution-policies" help:"IAM policy ARNs for CFN execution role."`
	PermissionsBoundary string `name:"permissions-boundary" help:"IAM permissions boundary for bootstrap roles."`
}

func (c *BootstrapCmd) Run(cfg *projcfg.Config) error {
	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	cdkArgs := cfg.Cdk.CdkArgs(cctx.Qualifier)
	args := make([]string, 0, 1+len(cdkArgs)+4)
	args = append(args, "bootstrap")
	args = append(args, cdkArgs...)
	if c.ExecutionPolicies != "" {
		args = append(args, "--cloudformation-execution-policies", c.ExecutionPolicies)
	}
	if c.PermissionsBoundary != "" {
		args = append(args, "--custom-permissions-boundary", c.PermissionsBoundary)
	}
	return cmdexec.Run(context.Background(), cfg.CdkDir(), "cdk", args...)
}
