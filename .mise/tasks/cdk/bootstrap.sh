#!/usr/bin/env bash
#MISE description="Bootstrap CDK in the current AWS account/region"

set -euo pipefail

cd infra/cdk/cdk
cdk bootstrap
