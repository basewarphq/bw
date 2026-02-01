#!/usr/bin/env bash
#MISE description="Deploy CDK stacks for a deployment"
#USAGE arg "deployment" help="Deployment name (e.g., Staging, Production)"

set -euo pipefail

cd infra/cdk/cdk
cdk deploy "bwapp*${usage_deployment:?}"
