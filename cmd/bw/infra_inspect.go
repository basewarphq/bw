package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type InfraInspectCmd struct {
	Deployment string   `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
	Lens       []string `short:"l" help:"Run specific inspections (e.g. endpoints, logs, 1password-sync)."`
}

func (c *InfraInspectCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := context.Background()
	if c.Deployment != "" {
		ctx = tool.WithDeployment(ctx, c.Deployment)
	}
	if len(c.Lens) > 0 {
		ctx = tool.WithInspectSelection(ctx, c.Lens)
	}
	g, err := dag.Build(cfg.Projects, reg, cfg, []tool.Step{tool.StepInspect})
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g, cliReporter{})
}
