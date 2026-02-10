package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cfndeploy"
	"github.com/basewarphq/bw/cmd/internal/cfnparams"
	"github.com/basewarphq/bw/cmd/internal/cfnpatch"
	"github.com/basewarphq/bw/cmd/internal/cfnread"
	"github.com/basewarphq/bw/cmd/internal/cfnvalidate"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
	"github.com/cockroachdb/errors"
)

const devSlotExpirationDays = 7

type BootstrapCmd struct {
	Profile             string `help:"AWS profile to use for bootstrap (requires admin permissions)."`
	ExecutionPolicies   string `name:"execution-policies" help:"IAM policy ARNs for CFN execution role."`
	PermissionsBoundary string `name:"permissions-boundary" help:"IAM permissions boundary for bootstrap roles."`
}

func (c *BootstrapCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	profile := c.Profile
	if profile == "" {
		profile = cfg.Cdk.Profile
	}

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	executionPolicies, permissionsBoundary, err := c.resolveBootstrapFlags(ctx, cfg, cctx, profile)
	if err != nil {
		return err
	}

	templatePath, err := patchedBootstrapTemplate(ctx, cfg)
	if err != nil {
		return err
	}
	defer os.Remove(templatePath)

	cdkArgs := cfg.Cdk.CdkArgs(cctx.Qualifier)
	args := make([]string, 0, 1+len(cdkArgs)+8)
	args = append(args, "bootstrap")
	args = append(args, cdkArgs...)
	args = append(args, "--template", templatePath)
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	if executionPolicies != "" {
		args = append(args, "--cloudformation-execution-policies", executionPolicies)
	}
	if permissionsBoundary != "" {
		args = append(args, "--custom-permissions-boundary", permissionsBoundary)
	}
	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", args...)
}

func (c *BootstrapCmd) resolveBootstrapFlags(
	ctx context.Context, cfg *projcfg.Config, cctx *cdkctx.CDKContext, profile string,
) (executionPolicies, permissionsBoundary string, err error) {
	if cfg.Cdk.PreBootstrap == nil {
		return c.ExecutionPolicies, c.PermissionsBoundary, nil
	}

	outputs, err := runPreBootstrap(ctx, cfg, cctx, profile)
	if err != nil {
		return "", "", err
	}

	executionPolicies = c.ExecutionPolicies
	if v := outputs["ExecutionPolicyArn"]; v != "" {
		if c.ExecutionPolicies != "" {
			return "", "", errors.New(
				"--execution-policies cannot be used when pre-bootstrap stack provides ExecutionPolicyArn",
			)
		}
		executionPolicies = v
	}

	permissionsBoundary = c.PermissionsBoundary
	if v := outputs["PermissionBoundaryName"]; v != "" {
		if c.PermissionsBoundary != "" {
			return "", "", errors.New(
				"--permissions-boundary cannot be used when pre-bootstrap stack provides PermissionBoundaryName",
			)
		}
		permissionsBoundary = v
	}

	return executionPolicies, permissionsBoundary, nil
}

func runPreBootstrap(
	ctx context.Context, cfg *projcfg.Config, cctx *cdkctx.CDKContext, profile string,
) (map[string]string, error) {
	pb := cfg.Cdk.PreBootstrap
	templatePath := filepath.Join(cfg.Root, pb.Template)

	if err := cfnvalidate.PreBootstrapTemplate(templatePath); err != nil {
		return nil, errors.Wrap(err, "validating pre-bootstrap template")
	}

	params, err := cfnparams.Resolve(pb.Parameters, cctx.ContextValues)
	if err != nil {
		return nil, errors.Wrap(err, "resolving pre-bootstrap parameters")
	}

	stackName := cctx.Qualifier + "-pre-bootstrap"

	fmt.Fprintf(os.Stderr, "Deploying pre-bootstrap stack %s...\n", stackName)
	if err := cfndeploy.Deploy(ctx, cfg.Root, profile, stackName, templatePath, params); err != nil {
		return nil, errors.Wrap(err, "deploying pre-bootstrap stack")
	}

	outputs, err := cfnread.StackOutputs(ctx, cctx.PrimaryRegion, profile, stackName)
	if err != nil {
		return nil, errors.Wrap(err, "reading pre-bootstrap stack outputs")
	}

	return outputs, nil
}

func patchedBootstrapTemplate(ctx context.Context, cfg *projcfg.Config) (string, error) {
	templateYAML, err := cmdexec.Output(ctx, cfg.CdkDir(), "cdk", "bootstrap", "--show-template")
	if err != nil {
		return "", errors.Wrap(err, "getting default bootstrap template")
	}

	patched, err := cfnpatch.AddDevSlotLifecycle([]byte(templateYAML), devSlotExpirationDays)
	if err != nil {
		return "", errors.Wrap(err, "patching bootstrap template")
	}

	tmpFile, err := os.CreateTemp("", "cdk-bootstrap-*.yaml")
	if err != nil {
		return "", errors.Wrap(err, "creating temp file for bootstrap template")
	}

	if _, err := tmpFile.Write(patched); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", errors.Wrap(err, "writing patched bootstrap template")
	}
	tmpFile.Close()

	return tmpFile.Name(), nil
}
