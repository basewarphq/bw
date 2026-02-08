package cdk

import (
	_ "embed"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdk1psync"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkcerts"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdns"
)

//go:embed 1password-saml-metadata.xml
var onePasswordSAMLMetadata string

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

	bwcdk1psync.NewProvider(stack, bwcdk1psync.ProviderProps{
		SAMLMetadataDocument: jsii.String(onePasswordSAMLMetadata),
	})

	return shared
}
