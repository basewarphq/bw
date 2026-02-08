package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type BootstrapCmd struct{}

func (c *BootstrapCmd) Run(cfg *projcfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.CdkDir(), "cdk", "bootstrap")
}
