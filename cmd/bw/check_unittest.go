package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type UnitTestCmd struct{}

func (c *UnitTestCmd) Run(cfg *wscfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.Root, "go", "test", "./...")
}
