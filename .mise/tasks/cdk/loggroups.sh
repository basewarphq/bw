#!/usr/bin/env bash
#MISE description="Show all CloudWatch log groups for a deployment"
#USAGE arg "deployment" help="Deployment name (e.g., Staging, Prod)"

set -euo pipefail

cd infra/cdk/cdk

# Map region identifiers back to AWS region codes
region_for_ident() {
	case "$1" in
	Euc1) echo "eu-central-1" ;;
	Euc2) echo "eu-central-2" ;;
	Euw1) echo "eu-west-1" ;;
	Euw2) echo "eu-west-2" ;;
	Euw3) echo "eu-west-3" ;;
	Eun1) echo "eu-north-1" ;;
	Eus1) echo "eu-south-1" ;;
	Eus2) echo "eu-south-2" ;;
	Use1) echo "us-east-1" ;;
	Use2) echo "us-east-2" ;;
	Usw1) echo "us-west-1" ;;
	Usw2) echo "us-west-2" ;;
	*) echo "" ;;
	esac
}

cdk list 2>/dev/null | grep "${usage_deployment:?}$" | while read -r stack; do
	# Extract region ident from stack name (e.g., bwappEuw1Stag -> Euw1)
	region_ident=$(echo "$stack" | sed -E 's/^bwapp([A-Z][a-z]+[0-9]).*/\1/')
	region=$(region_for_ident "$region_ident")

	echo "=== $stack ($region) ==="
	aws cloudformation describe-stacks \
		--no-cli-pager \
		--region "$region" \
		--stack-name "$stack" \
		--query "Stacks[0].Outputs[?contains(OutputKey, 'LogGroup')]" \
		--output table 2>/dev/null || echo "(not deployed)"
	echo ""
done
