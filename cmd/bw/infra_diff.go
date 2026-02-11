package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type InfraDiffCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
}

func (c *InfraDiffCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := context.Background()
	if c.Deployment != "" {
		ctx = tool.WithDeployment(ctx, c.Deployment)
	}
	g, err := dag.Build(cfg.Projects, reg, cfg, []tool.Step{tool.StepDiff})
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g, cliReporter{})
}
