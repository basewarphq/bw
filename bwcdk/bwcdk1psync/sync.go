// Package bwcdk1psync provides CDK constructs for setting up 1Password Environment
// sync to AWS Secrets Manager.
//
// The sync uses SAML-based authentication where 1Password's Confidential Computing
// platform assumes an IAM role to write secrets. This package creates the required
// AWS IAM resources; the 1Password UI is used to complete the integration setup.
//
// Architecture:
//   - SAML Provider: Created once per AWS account in the shared stack (global IAM resource)
//   - IAM Role: Created per deployment/environment with a unique SAML subject
//   - Secret: Created and managed by 1Password sync (not by CDK)
package bwcdk1psync

import (
	"encoding/xml"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bw/bwcdk/bwcdkparams"
	"github.com/basewarphq/bw/bwcdk/bwcdkutil"
)

const paramsNamespace = "1psync"

// SAMLProviderARNOutputKey is the CloudFormation output key for the SAML provider ARN.
const SAMLProviderARNOutputKey = "OnePasswordSAMLProviderARN"

// ProviderProps configures the SAML provider construct.
type ProviderProps struct {
	// SAMLMetadataDocument is the XML content downloaded from 1Password.
	// In 1Password: Developer > View Environments > [env] > Destinations > Configure AWS > Download SAML metadata
	SAMLMetadataDocument *string
}

// NewProvider creates the 1Password Secrets Sync SAML identity provider.
// This should be called in shared stacks for all regions.
//
// In the primary region: Creates the SAML provider and stores the ARN in SSM.
// In secondary regions: Looks up the ARN from primary and stores locally for deployment stacks.
//
// Panics if SAMLMetadataDocument is not valid XML or is still a placeholder.
func NewProvider(scope constructs.Construct, props ProviderProps) {
	scope = constructs.NewConstruct(scope, jsii.String("OnePasswordSyncProvider"))

	region := *awscdk.Stack_Of(scope).Region()

	if bwcdkutil.IsPrimaryRegion(scope, region) {
		validateSAMLMetadata(*props.SAMLMetadataDocument)

		provider := awsiam.NewSamlProvider(scope, jsii.String("SAMLProvider"), &awsiam.SamlProviderProps{
			Name:             jsii.String("1PasswordSecretsSync"),
			MetadataDocument: awsiam.SamlMetadataDocument_FromXml(props.SAMLMetadataDocument),
		})

		bwcdkparams.Store(scope, "SAMLProviderARNParam", paramsNamespace, "saml-provider-arn",
			provider.SamlProviderArn())

		awscdk.NewCfnOutput(awscdk.Stack_Of(scope), jsii.String(SAMLProviderARNOutputKey), &awscdk.CfnOutputProps{
			Value:       provider.SamlProviderArn(),
			Description: jsii.String("SAML Provider ARN - paste into 1Password 'SAML provider ARN' field"),
		})
	} else {
		// Look up from primary region and store locally for deployment stacks.
		providerArn := bwcdkparams.Lookup(scope, "LookupSAMLProviderARN",
			paramsNamespace, "saml-provider-arn", "saml-provider-arn-lookup")

		bwcdkparams.Store(scope, "SAMLProviderARNParam", paramsNamespace, "saml-provider-arn",
			providerArn)
	}
}

// RoleARNOutputKey returns the CloudFormation output key for a sync role ARN.
func RoleARNOutputKey(identifier string) string {
	return "OnePasswordSyncRoleARN" + identifier
}

// SecretNameOutputKey returns the CloudFormation output key for a sync secret name.
func SecretNameOutputKey(identifier string) string {
	return "OnePasswordSyncSecretName" + identifier
}

// secretName generates the standardized secret name using the qualifier from scope.
// Format: {qualifier}/{deployment}/{identifier} (all lowercase).
func secretName(scope constructs.Construct, deployment, identifier string) string {
	qualifier := bwcdkutil.Qualifier(scope)
	return strings.ToLower(qualifier) + "/" + strings.ToLower(deployment) + "/" + strings.ToLower(identifier)
}

// SyncRole provides access to the 1Password sync role and its associated secret.
type SyncRole interface {
	// SecretRef returns a reference to the synced secret.
	// Use this to grant read permissions and get the secret name for runtime lookup.
	SecretRef() SecretRef
}

// SyncRoleProps configures the IAM role for a 1Password Environment sync.
type SyncRoleProps struct {
	// Identifier distinguishes multiple sync roles (e.g., "Main", "Backend").
	// Used in resource names and SSM parameter paths.
	Identifier *string

	// SAMLSubject is the unique subject from 1Password for this Environment.
	// In 1Password: Developer > View Environments > [env] > Destinations > Configure AWS > Copy SAML subject
	SAMLSubject *string
}

// NewSyncRole creates an IAM role that allows 1Password to sync secrets to AWS Secrets Manager.
// Call this in each deployment stack that needs its own 1Password Environment.
// The role ARN is output for use in 1Password configuration.
//
// Since IAM roles are global (account-wide), the role is only created in the primary region.
// Secondary regions still return a valid SyncRole for accessing the secret.
//
// Panics if SAMLSubject is a placeholder or invalid.
func NewSyncRole(scope constructs.Construct, props SyncRoleProps) SyncRole {
	id := "OnePasswordSyncRole"
	if props.Identifier != nil {
		id = "OnePasswordSyncRole" + *props.Identifier
	}
	scope = constructs.NewConstruct(scope, jsii.String(id))

	deployment := bwcdkutil.DeploymentIdent(scope)
	ident := ""
	if props.Identifier != nil {
		ident = *props.Identifier
	}

	syncRoleResult := &syncRole{
		secretRef: &secretRef{
			scope:      scope,
			secretName: secretName(scope, deployment, ident),
		},
	}

	region := *awscdk.Stack_Of(scope).Region()
	if !bwcdkutil.IsPrimaryRegion(scope, region) {
		return syncRoleResult
	}

	validateSAMLSubject(*props.SAMLSubject)

	providerArn := bwcdkparams.LookupLocal(scope, paramsNamespace, "saml-provider-arn")

	samlProvider := awsiam.SamlProvider_FromSamlProviderArn(scope, jsii.String("SAMLProvider"), providerArn)

	// Trust policy follows AWS SAML federation docs:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_saml.html
	// 1Password requires "programmatic access only" with saml:sub condition.
	principal := awsiam.NewFederatedPrincipal(
		samlProvider.SamlProviderArn(),
		&map[string]any{
			"StringEquals": map[string]any{
				"SAML:aud": "https://signin.aws.amazon.com/saml",
				"SAML:sub": props.SAMLSubject,
			},
		},
		jsii.String("sts:AssumeRoleWithSAML"),
	)

	policyStatements := &[]awsiam.PolicyStatement{
		awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
			Sid:    jsii.String("SecretsManagerAccess"),
			Effect: awsiam.Effect_ALLOW,
			Actions: jsii.Strings(
				"secretsmanager:CreateSecret",
				"secretsmanager:UpdateSecret",
				"secretsmanager:DeleteSecret",
				"secretsmanager:GetSecretValue",
				"secretsmanager:PutSecretValue",
				"secretsmanager:DescribeSecret",
				"secretsmanager:TagResource",
				"secretsmanager:UntagResource",
			),
			Resources: jsii.Strings("*"),
		}),
	}

	// Build role name: 1PasswordSecretsSync{Deployment}{Identifier}
	roleName := "1PasswordSecretsSync" + deployment
	if props.Identifier != nil {
		roleName += *props.Identifier
	}

	role := awsiam.NewRole(scope, jsii.String("Role"), &awsiam.RoleProps{
		RoleName:  jsii.String(roleName),
		AssumedBy: principal,
		InlinePolicies: &map[string]awsiam.PolicyDocument{
			"SecretsManagerSync": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: policyStatements,
			}),
		},
	})

	outputKey := RoleARNOutputKey("")
	if props.Identifier != nil {
		outputKey = RoleARNOutputKey(*props.Identifier)
	}

	awscdk.NewCfnOutput(awscdk.Stack_Of(scope), jsii.String(outputKey), &awscdk.CfnOutputProps{
		Value:       role.RoleArn(),
		Description: jsii.String("IAM Role ARN - paste into 1Password 'IAM role ARN' field"),
	})

	// Output the standardized secret name
	secretNameOutputKey := SecretNameOutputKey(deployment + ident)
	sn := secretName(scope, deployment, ident)

	awscdk.NewCfnOutput(awscdk.Stack_Of(scope), jsii.String(secretNameOutputKey), &awscdk.CfnOutputProps{
		Value:       jsii.String(sn),
		Description: jsii.String("Secret name - paste into 1Password 'Target secret name' field"),
	})

	return syncRoleResult
}

type syncRole struct {
	secretRef SecretRef
}

func (s *syncRole) SecretRef() SecretRef {
	return s.secretRef
}

// SecretRef provides safe access to a 1Password-synced secret.
// It only exposes operations that work before the secret exists:
//   - GrantRead: Creates IAM policy (doesn't require secret to exist)
//   - SecretName: Returns the name for runtime lookup
//
// The secret itself is created by 1Password sync, not by CDK.
// Lambda should fetch the secret at runtime using the SecretName.
type SecretRef interface {
	// GrantRead grants the grantee permission to read the secret.
	// This creates an IAM policy and does NOT require the secret to exist.
	GrantRead(grantee awsiam.IGrantable)

	// SecretName returns the secret name for runtime lookup.
	// Pass this to Lambda as an environment variable.
	SecretName() *string
}

type secretRef struct {
	scope      constructs.Construct
	secretName string
}

func (s *secretRef) GrantRead(grantee awsiam.IGrantable) {
	stack := awscdk.Stack_Of(s.scope)

	// Build ARN pattern for the secret.
	// AWS adds a 6-character random suffix to secret ARNs, so we use "*" wildcard.
	// Format: arn:aws:secretsmanager:{region}:{account}:secret:{name}-*
	secretArnPattern := awscdk.Fn_Join(jsii.String(""), &[]*string{
		jsii.String("arn:aws:secretsmanager:"),
		stack.Region(),
		jsii.String(":"),
		stack.Account(),
		jsii.String(":secret:"),
		jsii.String(s.secretName),
		jsii.String("-*"),
	})

	grantee.GrantPrincipal().AddToPrincipalPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Effect:    awsiam.Effect_ALLOW,
		Actions:   jsii.Strings("secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"),
		Resources: &[]*string{secretArnPattern},
	}))
}

func (s *secretRef) SecretName() *string {
	return jsii.String(s.secretName)
}

func validateSAMLMetadata(doc string) {
	trimmed := strings.TrimSpace(doc)

	if trimmed == "" {
		panic("bwcdk1psync: SAMLMetadataDocument is empty")
	}

	if strings.HasPrefix(trimmed, "<!--") {
		panic("bwcdk1psync: SAMLMetadataDocument appears to be a placeholder comment. " +
			"Download the SAML metadata: Developer > View Environments > [env] > " +
			"Destinations > Configure AWS > Download SAML metadata")
	}

	if !strings.HasPrefix(trimmed, "<?xml") && !strings.HasPrefix(trimmed, "<") {
		panic("bwcdk1psync: SAMLMetadataDocument does not appear to be valid XML")
	}

	var probe struct{}
	if err := xml.Unmarshal([]byte(doc), &probe); err != nil {
		panic("bwcdk1psync: SAMLMetadataDocument is not valid XML: " + err.Error())
	}
}

func validateSAMLSubject(subject string) {
	trimmed := strings.TrimSpace(subject)

	if trimmed == "" {
		panic("bwcdk1psync: SAMLSubject is empty. " +
			"Copy the SAML subject: Developer > View Environments > [env] > " +
			"Destinations > Configure AWS > Copy SAML subject")
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "todo") {
		panic("bwcdk1psync: SAMLSubject is still a placeholder ('" + subject + "'). " +
			"Copy the SAML subject: Developer > View Environments > [env] > " +
			"Destinations > Configure AWS > Copy SAML subject")
	}

	// 1Password SAML subjects are uppercase alphanumeric, typically 26 characters
	if len(trimmed) < 10 {
		panic("bwcdk1psync: SAMLSubject appears invalid (too short): '" + subject + "'. " +
			"Expected format like 'IH75D4N7CP6JCAEATQMBNETCHQ'")
	}
}
