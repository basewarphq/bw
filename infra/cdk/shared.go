package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	bwcdkcerts "github.com/basewarphq/bwapp/bwcdk/agcdkcerts"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdns"
)

type Shared struct {
	DNS          bwcdkdns.DNS
	Certificates bwcdkcerts.Certificates
}

func NewShared(stack awscdk.Stack) *Shared {
	dns := bwcdkdns.New(stack, bwcdkdns.Props{})
	shared := &Shared{
		DNS: dns,
		Certificates: bwcdkcerts.New(stack, bwcdkcerts.Props{
			HostedZone: dns.HostedZone(),
		}),
	}

	return shared
}
