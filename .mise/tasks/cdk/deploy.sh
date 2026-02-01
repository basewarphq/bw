#!/usr/bin/env bash
#MISE description="Deploy CDK stacks for a deployment"
#USAGE arg "deployment" help="Deployment name (e.g., Staging, Production)"
#USAGE flag "--hotswap" help="Enable CDK hotswap deployment for faster iterations"

set -euo pipefail

hotswap_flag=""
if [[ "${usage_hotswap:-false}" == "true" ]]; then
	hotswap_flag="--hotswap"
fi

cd infra/cdk/cdk
cdk deploy --require-approval "never" $hotswap_flag "bwapp*${usage_deployment:?}"
