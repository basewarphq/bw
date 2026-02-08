package main

import (
	"context"
	"fmt"
	"os"

	"github.com/basewarphq/bwapp/cmd/internal/cdkctx"
	"github.com/basewarphq/bwapp/cmd/internal/devslot"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type SlotReleaseCmd struct{}

func (c *SlotReleaseCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	claim, err := devslot.ReadClaimFile(cfg.Root)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	accountID, err := devslot.AccountID(ctx)
	if err != nil {
		return err
	}

	store := devslot.NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)

	if err := store.Release(ctx, claim.Slot, claim.Token); err != nil {
		return err
	}

	if err := devslot.RemoveClaimFile(cfg.Root); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Released %s\n", claim.Slot)
	return nil
}
