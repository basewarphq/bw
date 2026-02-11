package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type InfraBootstrapCmd struct {
	Profile             string `help:"AWS profile to use for bootstrap (requires admin permissions)."`
	ExecutionPolicies   string `name:"execution-policies" help:"IAM policy ARNs for CFN execution role."`
	PermissionsBoundary string `name:"permissions-boundary" help:"IAM permissions boundary for bootstrap roles."`
}

func (c *InfraBootstrapCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := context.Background()
	ctx = tool.WithBootstrapOptions(ctx, tool.BootstrapOptions{
		Profile:             c.Profile,
		ExecutionPolicies:   c.ExecutionPolicies,
		PermissionsBoundary: c.PermissionsBoundary,
	})
	g, err := dag.Build(cfg.Projects, reg, cfg, []tool.Step{tool.StepBootstrap})
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g, cliReporter{})
}
