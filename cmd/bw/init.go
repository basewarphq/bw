package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type InitCmd struct{}

func (c *InitCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()
	return cmdexec.Run(ctx, cfg.Root, "op", "inject", "-i", ".env.tpl", "-o", ".env", "-f")
}
