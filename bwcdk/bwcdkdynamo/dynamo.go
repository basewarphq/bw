// Package bwcdkdynamo provides a reusable cross-region DynamoDB Global Table construct
// for multi-region CDK deployments.
//
// The construct creates a DynamoDB Global Table in the primary region with automatic
// replication to all secondary regions. It uses a single-table design with partition
// key (pk), sort key (sk), and two global secondary indexes (gsi1, gsi2).
package bwcdkdynamo

import (
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/basewarphq/bw/bwcdk/bwcdkparams"
	"github.com/basewarphq/bw/bwcdk/bwcdkutil"
)

const paramsNamespace = "dynamo"

// Dynamo provides access to a DynamoDB Global Table that works across regions.
type Dynamo interface {
	// Table returns the DynamoDB table.
	// In the primary region, this is the actual table.
	// In secondary regions, this is a reference to the replicated table.
	Table() awsdynamodb.ITableV2

	// GrantReadData grants read-only permissions to the table and its indexes.
	GrantReadData(grantee awsiam.IGrantable)

	// GrantReadWriteData grants read/write permissions to the table and its indexes.
	GrantReadWriteData(grantee awsiam.IGrantable)
}

// Props configures the Dynamo construct.
type Props struct {
	// Identifier distinguishes this table from others in the same deployment.
	// Used in resource names and SSM parameter paths.
	// Example: "main" produces table name "{qualifier}-{deployment}-main-table".
	Identifier *string
}

type dynamo struct {
	table      awsdynamodb.ITableV2
	identifier string
}

// New creates a Dynamo construct that manages a DynamoDB Global Table.
//
// In the primary region: Creates a new Global Table with replicas in all
// secondary regions and stores the table name in SSM Parameter Store.
//
// In secondary regions: Looks up the table name from SSM and creates a reference
// to the replicated table.
func New(scope constructs.Construct, props Props) Dynamo {
	identifier := "main"
	if props.Identifier != nil && *props.Identifier != "" {
		identifier = *props.Identifier
	}

	constructID := "Dynamo" + bwcdkutil.ResourceName(scope, identifier, bwcdkutil.CasingCamel)
	scope = constructs.NewConstruct(scope, jsii.String(constructID))
	con := &dynamo{identifier: identifier}

	region := *awscdk.Stack_Of(scope).Region()
	tableName := bwcdkutil.ResourceName(scope, identifier+"-table", bwcdkutil.CasingKebab)
	deploymentIdent := strings.ToLower(bwcdkutil.DeploymentIdent(scope))
	paramName := deploymentIdent + "/" + identifier + "/table-name"

	if bwcdkutil.IsPrimaryRegion(scope, region) {
		cfg := bwcdkutil.ConfigFromScope(scope)
		replicas := buildReplicas(cfg.SecondaryRegions)

		table := awsdynamodb.NewTableV2(scope, jsii.String("Table"), &awsdynamodb.TablePropsV2{
			TableName:     jsii.String(tableName),
			PartitionKey:  &awsdynamodb.Attribute{Name: jsii.String("pk"), Type: awsdynamodb.AttributeType_STRING},
			SortKey:       &awsdynamodb.Attribute{Name: jsii.String("sk"), Type: awsdynamodb.AttributeType_STRING},
			Billing:       awsdynamodb.Billing_OnDemand(nil),
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
			Replicas:      &replicas,
			PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
				PointInTimeRecoveryEnabled: jsii.Bool(true),
			},
			GlobalSecondaryIndexes: &[]*awsdynamodb.GlobalSecondaryIndexPropsV2{
				{
					IndexName:    jsii.String("gsi1"),
					PartitionKey: &awsdynamodb.Attribute{Name: jsii.String("gsi1pk"), Type: awsdynamodb.AttributeType_STRING},
					SortKey:      &awsdynamodb.Attribute{Name: jsii.String("gsi1sk"), Type: awsdynamodb.AttributeType_STRING},
				},
				{
					IndexName:    jsii.String("gsi2"),
					PartitionKey: &awsdynamodb.Attribute{Name: jsii.String("gsi2pk"), Type: awsdynamodb.AttributeType_STRING},
					SortKey:      &awsdynamodb.Attribute{Name: jsii.String("gsi2sk"), Type: awsdynamodb.AttributeType_STRING},
				},
			},
		})
		con.table = table

		bwcdkparams.Store(scope, "TableNameParam", paramsNamespace, paramName, jsii.String(tableName))
	} else {
		tableNameLookup := bwcdkparams.Lookup(scope, "LookupTableName",
			paramsNamespace, paramName, identifier+"-table-name-lookup")

		con.table = awsdynamodb.TableV2_FromTableName(scope, jsii.String("Table"), tableNameLookup)
	}

	return con
}

// LookupDynamo retrieves a DynamoDB table from SSM Parameter Store.
// Use this to get a table reference without creating cross-stack dependencies.
func LookupDynamo(scope constructs.Construct, identifier *string) awsdynamodb.ITableV2 {
	ident := "main"
	if identifier != nil && *identifier != "" {
		ident = *identifier
	}

	deploymentIdent := strings.ToLower(bwcdkutil.DeploymentIdent(scope))
	paramName := deploymentIdent + "/" + ident + "/table-name"
	tableName := bwcdkparams.LookupLocal(scope, paramsNamespace, paramName)

	lookupID := "LookupDynamo"
	if identifier != nil && *identifier != "" {
		lookupID = "LookupDynamo" + *identifier
	}

	return awsdynamodb.TableV2_FromTableName(scope, jsii.String(lookupID), tableName)
}

func (d *dynamo) Table() awsdynamodb.ITableV2 {
	return d.table
}

func (d *dynamo) GrantReadData(grantee awsiam.IGrantable) {
	d.table.GrantReadData(grantee)

	indexArn := jsii.Sprintf("%s/index/*", *d.table.TableArn())
	awsiam.Grant_AddToPrincipal(&awsiam.GrantOnPrincipalOptions{
		Grantee:      grantee,
		ResourceArns: &[]*string{indexArn},
		Actions: &[]*string{
			jsii.String("dynamodb:Query"),
			jsii.String("dynamodb:Scan"),
			jsii.String("dynamodb:GetItem"),
			jsii.String("dynamodb:BatchGetItem"),
			jsii.String("dynamodb:ConditionCheckItem"),
		},
	})
}

func (d *dynamo) GrantReadWriteData(grantee awsiam.IGrantable) {
	d.table.GrantReadWriteData(grantee)

	indexArn := jsii.Sprintf("%s/index/*", *d.table.TableArn())
	awsiam.Grant_AddToPrincipal(&awsiam.GrantOnPrincipalOptions{
		Grantee:      grantee,
		ResourceArns: &[]*string{indexArn},
		Actions: &[]*string{
			jsii.String("dynamodb:Query"),
			jsii.String("dynamodb:Scan"),
			jsii.String("dynamodb:GetItem"),
			jsii.String("dynamodb:BatchGetItem"),
			jsii.String("dynamodb:ConditionCheckItem"),
		},
	})
}

func buildReplicas(secondaryRegions []string) []*awsdynamodb.ReplicaTableProps {
	replicas := make([]*awsdynamodb.ReplicaTableProps, 0, len(secondaryRegions))
	for _, region := range secondaryRegions {
		replicas = append(replicas, &awsdynamodb.ReplicaTableProps{
			Region: jsii.String(region),
			PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
				PointInTimeRecoveryEnabled: jsii.Bool(true),
			},
		})
	}
	return replicas
}
