package main

type SlotsCmd struct {
	Claim   SlotClaimCmd   `cmd:"" help:"Claim a free dev deployment slot."`
	Release SlotReleaseCmd `cmd:"" help:"Release a claimed dev slot."`
	Status  SlotStatusCmd  `cmd:"" help:"Show status of all dev slots."`
}
