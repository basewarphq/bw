package main

import (
	"github.com/basewarphq/bwapp/infra/cdk"

	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

const projectPrefix = "bwapp"

func main() {
	defer jsii.Close()
	app := awscdk.NewApp(nil)

	bwcdkutil.SetupApp(app, bwcdkutil.AppConfig{
		Prefix:                projectPrefix+"-",
		DeployersGroup:        projectPrefix+"-deployers",
		RestrictedDeployments: []string{"Stag", "Prod"},
	},
		cdk.NewShared,
		cdk.NewDeployment,
	)

	app.Synth(nil)
}
