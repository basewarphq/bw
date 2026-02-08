---
name: build-cdk-construct
description: Build reusable AWS CDK constructs with cross-region SSM parameter patterns. Use when creating CDK constructs, per-deployment resources, shared infrastructure, or multi-region deployments.
---

# CDK Constructs

Guide for creating reusable, cross-region AWS CDK constructs using SSM Parameter Store for cross-stack communication.

## Stack Architecture

### Stack Types

**Shared Stacks** - Created per region, contain infrastructure shared across deployments:
- Route53 hosted zones, ACM certificates, VPCs, etc.
- Created by `SharedConstructor` in `bwcdkutil.SetupApp`
- Primary shared stack deploys first, secondary shared stacks depend on it

**Deployment Stacks** - Created per deployment per region:
- Application-specific resources (Lambda, API Gateway, DynamoDB, etc.)
- Created by `DeploymentConstructor` with a `deploymentIdent` parameter
- Depend on ALL shared stacks (deploy after shared infrastructure is ready)
- Primary deployment deploys first, secondary deployments depend on it

### Deployment Identifiers

Deployments represent environments (e.g., "Prod", "Staging", "Dev"):
- At least one deployment must be named "Prod"
- `deploymentIdent` is passed to `DeploymentConstructor`
- Use for resource naming, subdomain prefixes, configuration selection

### Deployment Order

```
Phase 1: Shared Infrastructure
  Primary Shared Stack
       ↓
  Secondary Shared Stacks (parallel)

Phase 2: Deployments (per deployment)
  Primary Deployment Stack
       ↓
  Secondary Deployment Stacks (parallel)
```

## Construct Structure

Every construct follows this pattern:

```go
package bwcdkfoo

import (
    "github.com/aws/constructs-go/constructs/v10"
    "github.com/aws/jsii-runtime-go"
    agcdkparams "github.com/basewarphq/bwapp/bwcdk/bwcdkparams"
)

const paramsNamespace = "foo"

// Foo provides access to the Foo resource.
type Foo interface {
    // Resource returns the underlying CDK resource.
    Resource() SomeIResource
}

// Props configures the Foo construct.
type Props struct {
    // Name is required. Description of what it does.
    Name *string
    // Optional field with default behavior documented.
    Optional *string
}

type foo struct {
    resource SomeIResource
}

// New creates a Foo construct.
func New(scope constructs.Construct, props Props) Foo {
    scope = constructs.NewConstruct(scope, jsii.String("Foo"))
    con := &foo{}
    
    // Implementation here
    
    return con
}

func (f *foo) Resource() SomeIResource {
    return f.resource
}

// LookupFoo retrieves the Foo resource from SSM Parameter Store.
// Use this to get a reference without creating cross-stack dependencies.
func LookupFoo(scope constructs.Construct) SomeIResource {
    arn := agcdkparams.LookupLocal(scope, paramsNamespace, "resource-arn")
    return SomeResource_FromArn(scope, jsii.String("LookupFoo"), arn)
}
```

## Package Naming

Name construct packages after the AWS service they encapsulate. Add a qualifying prefix when the construct implements a specific pattern or integration.

**Examples:**
- Generic Lambda construct → `{prefix}lambda`
- Lambda using AWS LWA specifically → `{prefix}lwalambda`
- Generic S3 construct → `{prefix}s3`

Use the specific name only when implementation details meaningfully distinguish it from a generic wrapper.

## Cross-Stack References via SSM

**Never use CDK cross-stack references** (passing constructs between stacks). They create tight coupling and deployment issues.

### Writing Values (Shared Stacks)

```go
agcdkparams.Store(scope, "ResourceArnParam", paramsNamespace, "resource-arn",
    resource.Arn())
```

### Reading Values - Same Region (Deployment Stacks)

```go
arn := agcdkparams.LookupLocal(scope, paramsNamespace, "resource-arn")
resource := SomeResource_FromArn(scope, jsii.String("LookupResource"), arn)
```

### Reading Values - Cross Region (Secondary Shared Stacks)

```go
arn := agcdkparams.Lookup(scope, "LookupResourceArn",
    paramsNamespace, "resource-arn", "resource-arn-lookup")
```

The `physicalID` (last parameter) must be unique and stable for the custom resource.

## Global vs Regional Resources

### Global Resources (e.g., Route53 Hosted Zone)

Created once in primary region, referenced everywhere:

```go
region := *awscdk.Stack_Of(scope).Region()

if bwcdkutil.IsPrimaryRegion(scope, region) {
    // Create the resource
    hostedZone := awsroute53.NewHostedZone(scope, jsii.String("HostedZone"),
        &awsroute53.HostedZoneProps{ZoneName: zoneName})
    con.hostedZone = hostedZone
    
    // Store for cross-region access
    agcdkparams.Store(scope, "HostedZoneIDParam", paramsNamespace, "hosted-zone-id",
        hostedZone.HostedZoneId())
} else {
    // Look up from primary region
    hostedZoneID := agcdkparams.Lookup(scope, "LookupHostedZoneID",
        paramsNamespace, "hosted-zone-id", "hosted-zone-id-lookup")
    
    // Store locally for deployment stacks in this region
    agcdkparams.Store(scope, "HostedZoneIDParam", paramsNamespace, "hosted-zone-id",
        hostedZoneID)
    
    // Reconstruct the reference
    con.hostedZone = awsroute53.HostedZone_FromHostedZoneAttributes(scope,
        jsii.String("HostedZone"), &awsroute53.HostedZoneAttributes{
            HostedZoneId: hostedZoneID,
            ZoneName:     zoneName,
        })
}
```

### Regional Resources (e.g., ACM Certificate)

Created independently in each region:

```go
// No primary/secondary branching needed
certificate := awscertificatemanager.NewCertificate(scope,
    jsii.String("WildcardCertificate"),
    &awscertificatemanager.CertificateProps{
        DomainName: jsii.String("*." + *props.HostedZone.ZoneName()),
        Validation: awscertificatemanager.CertificateValidation_FromDns(props.HostedZone),
    })

agcdkparams.Store(scope, "CertificateArnParam", paramsNamespace, "wildcard-cert-arn",
    certificate.CertificateArn())
```

## Multiple Instantiation Support

When a construct may be created multiple times per stack, parameterize IDs:

```go
// Props includes an identifier
type Props struct {
    // Identifier distinguishes multiple instances. Used in resource names and SSM paths.
    Identifier *string
}

func New(scope constructs.Construct, props Props) Foo {
    id := "Foo"
    if props.Identifier != nil {
        id = "Foo" + *props.Identifier
    }
    scope = constructs.NewConstruct(scope, jsii.String(id))
    
    // Use identifier in SSM parameter names
    paramName := "resource-arn"
    if props.Identifier != nil {
        paramName = *props.Identifier + "-resource-arn"
    }
    
    agcdkparams.Store(scope, "ResourceArnParam", paramsNamespace, paramName,
        resource.Arn())
    
    return con
}

// Lookup also needs the identifier
func LookupFoo(scope constructs.Construct, identifier *string) SomeIResource {
    paramName := "resource-arn"
    if identifier != nil {
        paramName = *identifier + "-resource-arn"
    }
    arn := agcdkparams.LookupLocal(scope, paramsNamespace, paramName)
    
    lookupID := "LookupFoo"
    if identifier != nil {
        lookupID = "LookupFoo" + *identifier
    }
    return SomeResource_FromArn(scope, jsii.String(lookupID), arn)
}
```

## SSM Parameter Naming

Parameters follow the pattern: `/{qualifier}/{namespace}/{name}`

- **qualifier**: From CDK context, identifies the app (e.g., "bwapp")
- **namespace**: Groups related parameters (e.g., "dns", "certs")
- **name**: Specific parameter (e.g., "hosted-zone-id", "wildcard-cert-arn")

Use `bwcdkparams.ParameterName()` to generate consistent paths.

### Per-Deployment Parameter Scoping

Constructs used in deployment stacks must include the deployment identifier in their SSM parameter `name` to prevent collisions across deployments (e.g., Prod vs Dev01). Scope at the **call site** in the construct, not in `bwcdkparams.ParameterName()` — that function is also used by shared stacks where no deployment identifier exists.

```go
deploymentIdent := strings.ToLower(bwcdkutil.DeploymentIdent(scope))
paramName := deploymentIdent + "/" + identifier + "/table-name"
bwcdkparams.Store(scope, "TableNameParam", paramsNamespace, paramName, jsii.String(tableName))
```

This produces paths like `/{qualifier}/{namespace}/{deployment}/{identifier}/table-name`, while shared-stack parameters remain at `/{qualifier}/{namespace}/{name}`.

## Resource Reconstruction

When looking up resources, reconstruct them using `FromXxx()` methods:

| Resource Type | Reconstruction Method |
|--------------|----------------------|
| Hosted Zone | `HostedZone_FromHostedZoneAttributes` |
| Certificate | `Certificate_FromCertificateArn` |
| VPC | `Vpc_FromVpcAttributes` |
| Security Group | `SecurityGroup_FromSecurityGroupId` |
| Lambda | `Function_FromFunctionArn` |
| DynamoDB Table | `Table_FromTableArn` |

Store the minimum needed to reconstruct (usually ID or ARN).

## CloudFormation Outputs

Export values for manual inspection when useful:

```go
awscdk.NewCfnOutput(awscdk.Stack_Of(scope), jsii.String("HostedZoneNameServers"),
    &awscdk.CfnOutputProps{
        Value:       awscdk.Fn_Join(jsii.String(","), hostedZone.HostedZoneNameServers()),
        Description: jsii.String("Comma-separated list of NS records for DNS delegation"),
    })
```

Use a `const` for the output key if consumers need to reference it programmatically.

## Log Groups

**Always use `bwcdkloggroup` for creating CloudWatch Log Groups.** This ensures:
- Consistent retention (ONE_WEEK) and removal policy (DESTROY)
- Automatic CfnOutput export for CLI discoverability (query `*LogGroup` outputs)

```go
import "github.com/basewarphq/bwapp/bwcdk/bwcdkloggroup"

logGroup := bwcdkloggroup.New(scope, "MyServiceLogs", bwcdkloggroup.Props{
    Purpose: jsii.String("Lambda function logs"),
})

// Use logGroup.LogGroup() when passing to other constructs
lambda := awslambda.NewFunction(scope, jsii.String("Fn"), &awslambda.FunctionProps{
    LogGroup: logGroup.LogGroup(),
    // ...
})
```

The `id` parameter should be descriptive and unique within the stack (e.g., `"BackendCorebackLogs"`, `"ApiAccessLogs"`).

## Checklist for New Constructs

- [ ] Package with `bwcdk` prefix (e.g., `bwcdkfoo`)
- [ ] Package doc comment explaining purpose
- [ ] `const paramsNamespace` for SSM grouping
- [ ] Public interface type with getter methods
- [ ] Private concrete struct type
- [ ] `Props` struct with documented fields
- [ ] `New()` constructor returning interface
- [ ] `LookupXxx()` function for cross-stack access
- [ ] Primary/secondary region handling (if global resource)
- [ ] SSM storage for values needed by other stacks
- [ ] Identifier support (if multiple instances possible)

## Reference Examples

- Global resource: `bwcdk/bwcdkdns/dns.go`
- Regional resource: `bwcdk/agcdkcerts/certificates.go`
- Log groups: `bwcdk/bwcdkloggroup/loggroup.go`
- Parameters package: `bwcdk/bwcdkparams/params.go`
- App setup: `bwcdk/bwcdkutil/app.go`
- Config: `bwcdk/bwcdkutil/config.go`
