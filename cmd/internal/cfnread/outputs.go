package cfnread

import (
	"context"
	"encoding/json"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/cockroachdb/errors"
)

type describeStacksResponse struct {
	Stacks []struct {
		Outputs []struct {
			OutputKey   string `json:"OutputKey"`
			OutputValue string `json:"OutputValue"`
		} `json:"Outputs"`
	} `json:"Stacks"`
}

func StackOutputs(ctx context.Context, region, stackName string) (map[string]string, error) {
	out, err := cmdexec.Output(ctx, "/", "aws", "cloudformation", "describe-stacks",
		"--no-cli-pager",
		"--region", region,
		"--stack-name", stackName,
		"--output", "json",
	)
	if err != nil {
		return nil, errors.Wrapf(err, "describing stack %s in %s", stackName, region)
	}

	var resp describeStacksResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, errors.Wrapf(err, "parsing stack outputs for %s", stackName)
	}

	if len(resp.Stacks) == 0 {
		return nil, errors.Newf("stack %s not found in %s", stackName, region)
	}

	outputs := make(map[string]string, len(resp.Stacks[0].Outputs))
	for _, o := range resp.Stacks[0].Outputs {
		outputs[o.OutputKey] = o.OutputValue
	}
	return outputs, nil
}
