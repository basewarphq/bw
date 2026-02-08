package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
	"github.com/basewarphq/bwapp/cmd/internal/shellfiles"
)

type FmtCmd struct{}

func (c *FmtCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()
	if err := cmdexec.Run(ctx, cfg.Root, "golangci-lint", "fmt", "./..."); err != nil {
		return err
	}

	scripts, err := shellfiles.FindShellScripts(cfg.Root)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}

	args := append([]string{"-w"}, scripts...)
	return cmdexec.Run(ctx, cfg.Root, "shfmt", args...)
}
