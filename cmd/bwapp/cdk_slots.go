package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cdkctx"
	"github.com/basewarphq/bwapp/cmd/internal/devslot"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
	"github.com/cockroachdb/errors"
)

type SlotsCmd struct {
	Claim   SlotClaimCmd   `cmd:"" help:"Claim a free dev deployment slot."`
	Release SlotReleaseCmd `cmd:"" help:"Release a claimed dev slot."`
	Status  SlotStatusCmd  `cmd:"" help:"Show status of all dev slots."`
}

func ensureClaim(ctx context.Context, cfg *projcfg.Config) (*devslot.ClaimFile, error) {
	claim, err := devslot.ReadClaimFile(cfg.Root)
	if err != nil && !errors.Is(err, devslot.ErrNoClaim) {
		return nil, err
	}
	if claim != nil {
		touchSlotClaim(ctx, cfg, claim)
		return claim, nil
	}

	return newClaim(ctx, cfg)
}

func newClaim(ctx context.Context, cfg *projcfg.Config) (*devslot.ClaimFile, error) {
	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return nil, err
	}

	slots := cctx.DevSlots()
	if len(slots) == 0 {
		return nil, errors.New("no Dev* deployments defined in cdk.context.json")
	}

	token, err := devslot.GenerateToken()
	if err != nil {
		return nil, err
	}

	accountID, err := devslot.AccountID(ctx)
	if err != nil {
		return nil, err
	}

	store := devslot.NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)
	label := devslot.DefaultLabel(ctx)

	slot, err := devslot.ClaimFirstAvailable(ctx, store, slots, token, label)
	if err != nil {
		return nil, err
	}

	claim := &devslot.ClaimFile{Slot: slot, Token: token}
	if err := devslot.WriteClaimFile(cfg.Root, claim); err != nil {
		return nil, err
	}
	return claim, nil
}

func touchSlotClaim(
	ctx context.Context, cfg *projcfg.Config, claim *devslot.ClaimFile,
) {
	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return
	}
	accountID, err := devslot.AccountID(ctx)
	if err != nil {
		return
	}
	store := devslot.NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)
	_ = store.Touch(ctx, claim.Slot, claim.Token)
}
