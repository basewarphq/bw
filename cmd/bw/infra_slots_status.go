package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/devslot"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
	"github.com/cockroachdb/errors"
)

type InfraSlotStatusCmd struct{}

func (c *InfraSlotStatusCmd) Run(cfg *wscfg.Config) error {
	ctx := context.Background()

	dir, profile, err := cdkProjectDirAndProfile(cfg)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return err
	}

	slots := cctx.DevSlots()
	if len(slots) == 0 {
		return errors.New("no Dev* deployments defined in cdk.context.json")
	}

	accountID, err := devslot.AccountID(ctx, profile)
	if err != nil {
		return err
	}

	store := devslot.NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)

	statuses, err := store.ListAll(ctx, slots)
	if err != nil {
		return err
	}

	claim, _ := devslot.ReadClaimFile(dir)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLOT\tSTATUS\tLABEL\tLAST USED")
	for _, slot := range slots {
		info := statuses[slot]
		status := "free"
		if info != nil {
			status = "claimed"
			if claim != nil && claim.Slot == slot {
				status = "claimed (*)"
			}
		}
		if info == nil {
			fmt.Fprintf(w, "%s\t%s\t\t\n", slot, status)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				slot, status, info.Label, info.LastUsed)
		}
	}
	w.Flush()

	return nil
}
