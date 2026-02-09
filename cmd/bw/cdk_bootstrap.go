package main

import (
	"context"
	"os"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cfnpatch"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
	"github.com/cockroachdb/errors"
)

const devSlotExpirationDays = 7

type BootstrapCmd struct {
	ExecutionPolicies   string `name:"execution-policies" help:"IAM policy ARNs for CFN execution role."`
	PermissionsBoundary string `name:"permissions-boundary" help:"IAM permissions boundary for bootstrap roles."`
}

func (c *BootstrapCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	templatePath, err := patchedBootstrapTemplate(ctx, cfg)
	if err != nil {
		return err
	}
	defer os.Remove(templatePath)

	cdkArgs := cfg.Cdk.CdkArgs(cctx.Qualifier)
	args := make([]string, 0, 1+len(cdkArgs)+6)
	args = append(args, "bootstrap")
	args = append(args, cdkArgs...)
	args = append(args, "--template", templatePath)
	if c.ExecutionPolicies != "" {
		args = append(args, "--cloudformation-execution-policies", c.ExecutionPolicies)
	}
	if c.PermissionsBoundary != "" {
		args = append(args, "--custom-permissions-boundary", c.PermissionsBoundary)
	}
	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", args...)
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
