package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	bwcdkcerts "github.com/basewarphq/bwapp/bwcdk/agcdkcerts"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdns"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
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

	// Preserve exports from removed RestGateway construct.
	// TODO: Remove after all deployment stacks are updated.
	region := *stack.Region()
	if bwcdkutil.IsPrimaryRegion(stack, region) {
		bwcdkutil.PreserveExport(stack, "PreserveHostedZoneRef",
			*stack.StackName()+":ExportsOutputRefDNSHostedZone3D281E9A7300202C",
			dns.HostedZone().HostedZoneId())
	} else {
		bwcdkutil.PreserveExport(stack, "PreserveHostedZoneLookupRef",
			*stack.StackName()+":ExportsOutputFnGetAttDNSLookupHostedZoneIDE73E6A3EParameterValueED45C8A2",
			dns.HostedZone().HostedZoneId())
	}
	bwcdkutil.PreserveExport(stack, "PreserveCertificateRef",
		*stack.StackName()+":ExportsOutputRefCertificatesWildcardCertificate9430FE417295717B",
		shared.Certificates.WildcardCertificate().CertificateArn())

	return shared
}
