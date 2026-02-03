package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	bwcdkcerts "github.com/basewarphq/bwapp/bwcdk/agcdkcerts"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdns"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkdynamo"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkrestgateway"
)

func NewDeployment(stack awscdk.Stack, deploymentIdent string) {
	hostedZone := bwcdkdns.LookupHostedZone(stack, nil)
	certificate := bwcdkcerts.LookupCertificate(stack)
	dynamo := bwcdkdynamo.New(stack, bwcdkdynamo.Props{
		Identifier: jsii.String("main"),
	})

	gateway := bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry: jsii.String("../../../backend/cmd/coreback"),
		GatewayRoutes: &[]*bwcdkrestgateway.RouteConfig{
			{Path: jsii.String("/g/{proxy+}"), RequireAuth: jsii.Bool(true)},
		},
		HostedZone:  hostedZone,
		Certificate: certificate,
		Subdomain:   jsii.String("api"),
		Authorizer:  &bwcdkrestgateway.AuthorizerProps{},
		Environment: &map[string]*string{
			"MAIN_TABLE_NAME": dynamo.Table().TableName(),
		},
	})
	dynamo.GrantReadWriteData(gateway.Lambda().Function())
	dynamo.GrantReadData(gateway.AuthorizerLambda().Function())

	bwcdkrestgateway.New(stack, bwcdkrestgateway.Props{
		Entry: jsii.String("../../../console/cmd/coreconsole"),
		GatewayRoutes: &[]*bwcdkrestgateway.RouteConfig{
			{Path: jsii.String("/")},
			{Path: jsii.String("/c/{proxy+}"), RequireAuth: jsii.Bool(true)},
		},
		HostedZone:  hostedZone,
		Certificate: certificate,
		Subdomain:   jsii.String("console"),
		Authorizer:  &bwcdkrestgateway.AuthorizerProps{},
	})
}
