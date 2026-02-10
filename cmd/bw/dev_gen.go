package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type GenCmd struct{}

func (c *GenCmd) Run(cfg *wscfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.Root, "go", "generate", "./...")
}
