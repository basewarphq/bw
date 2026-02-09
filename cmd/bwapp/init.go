package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type InitCmd struct{}

func (c *InitCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()
	return cmdexec.Run(ctx, cfg.Root, "op", "inject", "-i", ".env.tpl", "-o", ".env", "-f")
}
