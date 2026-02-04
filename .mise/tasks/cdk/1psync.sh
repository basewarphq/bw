#!/usr/bin/env bash
#MISE description="Show 1Password sync configuration for a deployment"
#USAGE arg "deployment" help="Deployment name (e.g., Stag, Prod)"

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

# Find the primary region shared stack (first one listed)
shared_stack=$(cdk list 2>/dev/null | grep "Shared$" | head -1)
if [[ -z "$shared_stack" ]]; then
	echo "Error: No shared stack found"
	exit 1
fi

# Extract region from shared stack name
region_ident=$(echo "$shared_stack" | sed -E 's/^bwapp([A-Z][a-z]+[0-9]).*/\1/')
primary_region=$(region_for_ident "$region_ident")

# Find the primary region deployment stack
deployment_stack=$(cdk list 2>/dev/null | grep "${usage_deployment:?}$" | head -1)
if [[ -z "$deployment_stack" ]]; then
	echo "Error: No deployment stack found for ${usage_deployment}"
	exit 1
fi

echo "=== 1Password Sync Configuration for ${usage_deployment} ==="
echo ""

# Get SAML Provider ARN from shared stack
saml_arn=$(aws cloudformation describe-stacks \
	--no-cli-pager \
	--region "$primary_region" \
	--stack-name "$shared_stack" \
	--query "Stacks[0].Outputs[?OutputKey=='OnePasswordSAMLProviderARN'].OutputValue" \
	--output text 2>/dev/null || echo "")

# Get Role ARN from deployment stack
role_arn=$(aws cloudformation describe-stacks \
	--no-cli-pager \
	--region "$primary_region" \
	--stack-name "$deployment_stack" \
	--query "Stacks[0].Outputs[?contains(OutputKey, 'OnePasswordSyncRoleARN')].OutputValue" \
	--output text 2>/dev/null || echo "")

# Get Secret Name from deployment stack
secret_name=$(aws cloudformation describe-stacks \
	--no-cli-pager \
	--region "$primary_region" \
	--stack-name "$deployment_stack" \
	--query "Stacks[0].Outputs[?contains(OutputKey, 'OnePasswordSyncSecretName')].OutputValue" \
	--output text 2>/dev/null || echo "")

echo "Copy these values into 1Password:"
echo "  Developer > View Environments > [env] > Destinations > Configure AWS"
echo ""
echo "SAML provider ARN:"
echo "  ${saml_arn:-"(not found - deploy shared stack first)"}"
echo ""
echo "IAM role ARN:"
echo "  ${role_arn:-"(not found - deploy deployment stack first)"}"
echo ""
echo "Target region:"
echo "  ${primary_region}"
echo ""
echo "Target secret name:"
echo "  ${secret_name:-"(not found - deploy deployment stack first)"}"
