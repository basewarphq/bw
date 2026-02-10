package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/shellfiles"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type LintCmd struct{}

func (c *LintCmd) Run(cfg *wscfg.Config) error {
	ctx := context.Background()
	if err := cmdexec.Run(ctx, cfg.Root, "golangci-lint", "run", "./..."); err != nil {
		return err
	}

	scripts, err := shellfiles.FindShellScripts(cfg.Root)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}

	args := append([]string{}, scripts...)
	return cmdexec.Run(ctx, cfg.Root, "shellcheck", args...)
}
