package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type ReleaseCmd struct {
	DryRun bool `help:"Build release artifacts without pushing tags or publishing."`
}

func (c *ReleaseCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := context.Background()
	ctx = tool.WithReleaseOptions(ctx, tool.ReleaseOptions{
		DryRun: c.DryRun,
	})
	g, err := dag.Build(cfg.Projects, reg, cfg, tool.ReleaseSteps)
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g, cliReporter{})
}
