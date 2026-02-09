package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type UnitTestCmd struct{}

func (c *UnitTestCmd) Run(cfg *projcfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.Root, "go", "test", "./...")
}
