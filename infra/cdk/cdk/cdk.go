package main

import (
	"github.com/basewarphq/bwapp/infra/cdk"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
)

const projectIdent = "bwapp"

func main() {
	defer jsii.Close()
	app := awscdk.NewApp(nil)

	bwcdkutil.SetupApp(app, bwcdkutil.AppConfig{
		Prefix: projectIdent + "-",
	},
		cdk.NewShared,
		cdk.NewDeployment,
	)

	app.Synth(nil)
}
