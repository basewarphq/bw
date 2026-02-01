package cdk

import (
	"github.com/advdv/ago/agcdk/agcdksharedbase"
	"github.com/aws/aws-cdk-go/awscdk/v2"
)

type Shared struct {
	Base agcdksharedbase.SharedBase
}

func NewShared(stack awscdk.Stack) *Shared {
	shared := &Shared{}
	shared.Base = agcdksharedbase.New(stack, agcdksharedbase.Props{})

	if !shared.Base.IsValidated() {
		// Shared base not yet validated - only foundational resources created.
		// Deploy this first, complete validation steps (e.g., DNS delegation),
		// then set shared-base-validated=true.
		return shared
	}

	// Add shared resources that depend on DNS below

	return shared
}
