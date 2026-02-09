package main

import (
	"context"
	"fmt"
	"os"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/devslot"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type SlotReleaseCmd struct {
	Slot  string `help:"Name of the slot to release (default: this checkout's claimed slot)." short:"s"`
	Force bool   `help:"Force-release the slot even if it belongs to someone else." short:"f"`
}

func (c *SlotReleaseCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	accountID, err := devslot.AccountID(ctx)
	if err != nil {
		return err
	}

	store := devslot.NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)

	slot, token, isLocalClaim, err := c.resolveSlot(cfg)
	if err != nil {
		return err
	}

	if c.Force {
		if err := store.ForceRelease(ctx, slot); err != nil {
			return err
		}
	} else {
		if err := store.Release(ctx, slot, token); err != nil {
			return err
		}
	}

	if isLocalClaim {
		if err := devslot.RemoveClaimFile(cfg.Root); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Released %s\n", slot)
	return nil
}

func (c *SlotReleaseCmd) resolveSlot(cfg *projcfg.Config) (slot, token string, isLocal bool, err error) {
	claim, claimErr := devslot.ReadClaimFile(cfg.Root)

	if c.Slot == "" {
		if claimErr != nil {
			return "", "", false, claimErr
		}
		return claim.Slot, claim.Token, true, nil
	}

	if claim != nil && claim.Slot == c.Slot {
		return claim.Slot, claim.Token, true, nil
	}

	if !c.Force {
		return "", "", false, fmt.Errorf(
			"slot %s is not this checkout's claim; use --force to release it anyway", c.Slot,
		)
	}

	return c.Slot, "", false, nil
}
