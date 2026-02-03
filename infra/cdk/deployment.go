package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	bwcdkcerts "github.com/basewarphq/bwapp/bwcdk/agcdkcerts"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdns"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkrestgateway"
)

func NewDeployment(stack awscdk.Stack, deploymentIdent string) {
	hostedZone := bwcdkdns.LookupHostedZone(stack, nil)
	certificate := bwcdkcerts.LookupCertificate(stack)

	_ = bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry:        jsii.String("../../../backend/cmd/coreback"),
		PublicRoutes: jsii.Strings("/g/{proxy+}"),
		HostedZone:   hostedZone,
		Certificate:  certificate,
		Subdomain:    jsii.String("api"),
		Authorizer:   &bwcdkrestgateway.AuthorizerProps{},
	})
}
