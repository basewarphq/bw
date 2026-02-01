package main

import (
	"cdk/cdk"

	"github.com/advdv/ago/agcdkutil"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

const projectPrefix = "bwapp"

func main() {
	defer jsii.Close()
	app := awscdk.NewApp(nil)

	agcdkutil.SetupApp(app, agcdkutil.AppConfig{
		Prefix:                projectPrefix+"-",
		DeployersGroup:        projectPrefix+"-deployers",
		RestrictedDeployments: []string{"Stag", "Prod"},
	},
		cdk.NewShared,
		cdk.NewDeployment,
	)

	app.Synth(nil)
}
