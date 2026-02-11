package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type InfraDeployCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
	Hotswap    bool   `help:"Enable CDK hotswap deployment for faster iterations."`
}

func (c *InfraDeployCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := context.Background()
	if c.Deployment != "" {
		ctx = tool.WithDeployment(ctx, c.Deployment)
	}
	ctx = tool.WithDeployOptions(ctx, tool.DeployOptions{
		Hotswap: c.Hotswap,
	})
	g, err := dag.Build(cfg.Projects, reg, cfg, []tool.Step{tool.StepDeploy})
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g, cliReporter{})
}
