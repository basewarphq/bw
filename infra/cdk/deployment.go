package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkrestgateway"
)

func NewDeployment(stack awscdk.Stack, shared *Shared, deploymentIdent string) {
	_ = bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry:           jsii.String("../../../backend/cmd/coreback"),
		PublicRoutes:    jsii.Strings("/g/{proxy+}"),
		HostedZone:      shared.DNS.HostedZone(),
		Certificate:     shared.Certificates.WildcardCertificate(),
		Subdomain:       jsii.String("api"),
		DeploymentIdent: jsii.String(deploymentIdent),
	})
}
