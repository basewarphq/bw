package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type GenCmd struct{}

func (c *GenCmd) Run(cfg *projcfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.Root, "go", "generate", "./...")
}
