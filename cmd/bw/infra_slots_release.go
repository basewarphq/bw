package main

import (
	"context"
	"fmt"
	"os"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/devslot"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
	"github.com/cockroachdb/errors"
)

type InfraSlotReleaseCmd struct {
	Slot  string `help:"Name of the slot to release (default: this checkout's claimed slot)." short:"s"`
	Force bool   `help:"Force-release the slot even if it belongs to someone else." short:"f"`
}

func (c *InfraSlotReleaseCmd) Run(cfg *wscfg.Config) error {
	ctx := context.Background()

	dir, profile, err := infraProjectDirAndProfile(cfg)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return err
	}

	accountID, err := devslot.AccountID(ctx, profile)
	if err != nil {
		return err
	}

	store := devslot.NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)

	slot, token, isLocalClaim, err := c.resolveSlot(dir)
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
		if err := devslot.RemoveClaimFile(dir); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Released %s\n", slot)
	return nil
}

func (c *InfraSlotReleaseCmd) resolveSlot(projectDir string) (slot, token string, isLocal bool, err error) {
	claim, claimErr := devslot.ReadClaimFile(projectDir)

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
		return "", "", false, errors.Newf(
			"slot %s is not this checkout's claim; use --force to release it anyway", c.Slot,
		)
	}

	return c.Slot, "", false, nil
}
