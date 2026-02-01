package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
)

type Shared struct {
}

func NewShared(stack awscdk.Stack) *Shared {
	shared := &Shared{}
	return shared
}
