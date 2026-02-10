package cfndeploy

import (
	"context"
	"fmt"
	"sort"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
)

func Deploy(ctx context.Context, dir, profile, stackName, templatePath string, params map[string]string) error {
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
	if len(params) > 0 {
		args = append(args, "--parameter-overrides")
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			args = append(args, fmt.Sprintf("%s=%s", k, params[k]))
		}
	}
	return cmdexec.Run(ctx, dir, "aws", args...)
}
