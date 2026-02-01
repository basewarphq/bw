#!/usr/bin/env bash
#MISE description="Show CDK diff for a deployment"
#USAGE arg "deployment" help="Deployment name (e.g., Staging, Production)"

set -euo pipefail

cd infra/cdk/cdk
cdk diff "bwapp*${usage_deployment:?}"
