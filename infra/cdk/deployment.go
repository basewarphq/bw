package cdk

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdklwalambda"
)

func NewDeployment(stack awscdk.Stack, shared *Shared, deploymentIdent string) {
	_ = bwcdklwalambda.New(stack, bwcdklwalambda.Props{
		Entry: jsii.String("../../../backend/cmd/coreback"),
	})
}
