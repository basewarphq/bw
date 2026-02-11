package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/bincheck"
	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type PreflightCmd struct{}

func (c *PreflightCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := tool.WithBinChecker(context.Background(), bincheck.NewChecker())
	g, err := dag.Build(cfg.Projects, reg, cfg, tool.PreflightSteps)
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g, cliReporter{})
}
