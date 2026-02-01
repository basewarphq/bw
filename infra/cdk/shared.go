package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdns"
)

type Shared struct {
	DNS bwcdkdns.DNS
}

func NewShared(stack awscdk.Stack) *Shared {
	shared := &Shared{
		DNS: bwcdkdns.New(stack, bwcdkdns.Props{}),
	}
	return shared
}
