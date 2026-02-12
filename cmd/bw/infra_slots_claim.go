package main

import (
	"context"
	"fmt"
	"os"

	"github.com/basewarphq/bw/cmd/internal/devslot"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type InfraSlotClaimCmd struct{}

func (c *InfraSlotClaimCmd) Run(cfg *wscfg.Config) error {
	ctx := context.Background()

	dir, profile, err := infraProjectDirAndProfile(cfg)
	if err != nil {
		return err
	}

	claim, err := devslot.EnsureClaim(ctx, dir, profile)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, claim.Slot)
	return nil
}
