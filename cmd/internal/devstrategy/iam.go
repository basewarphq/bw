package devstrategy

import (
	"context"
	"strings"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/cockroachdb/errors"
)

func IAMDeployment(ctx context.Context, profile string) (string, error) {
	args := []string{
		"sts", "get-caller-identity",
		"--query", "Arn",
		"--output", "text",
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	out, err := cmdexec.Output(ctx, "/", "aws", args...)
	if err != nil {
		return "", errors.Wrap(err, "getting caller identity")
	}

	arn := strings.TrimSpace(out)
	parts := strings.Split(arn, "/")
	if len(parts) < 2 {
		return "", errors.Newf("unexpected ARN format: %s", arn)
	}
	username := parts[len(parts)-1]
	if username == "" {
		return "", errors.Newf("empty username in ARN: %s", arn)
	}

	return "Dev" + strings.ToUpper(username[:1]) + strings.ToLower(username[1:]), nil
}
