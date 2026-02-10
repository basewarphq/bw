package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type GenCmd struct{}

func (c *GenCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	ctx := context.Background()
	g, err := dag.Build(cfg.Projects, reg, cfg.Root, []tool.Step{tool.StepGen})
	if err != nil {
		return err
	}
	return dag.Execute(ctx, g)
}
