package cfndeploy

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
)

func Deploy(ctx context.Context, dir, profile, stackName, templatePath string) error {
	args := []string{
		"cloudformation", "deploy",
		"--stack-name", stackName,
		"--template-file", templatePath,
		"--capabilities", "CAPABILITY_NAMED_IAM",
		"--no-fail-on-empty-changeset",
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	return cmdexec.Run(ctx, dir, "aws", args...)
}
