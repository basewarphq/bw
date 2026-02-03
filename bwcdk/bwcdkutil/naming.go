package bwcdkutil

import (
	"fmt"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/iancoleman/strcase"
)

// Casing specifies how to format the identifier string.
type Casing int

const (
	// CasingCamel formats as CamelCase (e.g., "BwappStagApiGateway").
	CasingCamel Casing = iota
	// CasingLowerCamel formats as lowerCamelCase (e.g., "bwappStagApiGateway").
	CasingLowerCamel
	// CasingSnake formats as snake_case (e.g., "bwapp_stag_api_gateway").
	CasingSnake
	// CasingScreamingSnake formats as SCREAMING_SNAKE_CASE (e.g., "BWAPP_STAG_API_GATEWAY").
	CasingScreamingSnake
	// CasingKebab formats as kebab-case (e.g., "bwapp-stag-api-gateway").
	CasingKebab
	// CasingScreamingKebab formats as SCREAMING-KEBAB-CASE (e.g., "BWAPP-STAG-API-GATEWAY").
	CasingScreamingKebab
)

// ResourceName generates a resource identifier prefixed with the stack's qualifier
// and deployment identifier. The label is a free-form string that the caller provides.
//
// The format is: "{qualifier}-{deploymentIdent}-{label}" converted to the specified casing.
//
// For shared stacks (no deployment identifier), the format is: "{qualifier}-{label}".
//
// Examples with qualifier "bwapp", deployment "Stag", label "ApiGateway":
//   - CasingCamel:          "BwappStagApiGateway"
//   - CasingLowerCamel:     "bwappStagApiGateway"
//   - CasingSnake:          "bwapp_stag_api_gateway"
//   - CasingScreamingSnake: "BWAPP_STAG_API_GATEWAY"
//   - CasingKebab:          "bwapp-stag-api-gateway"
//   - CasingScreamingKebab: "BWAPP-STAG-API-GATEWAY"
func ResourceName(scope constructs.Construct, label string, casing Casing) string {
	qualifier := Qualifier(scope)
	deploymentIdent := DeploymentIdent(scope)

	var base string
	if deploymentIdent != "" {
		base = fmt.Sprintf("%s-%s-%s", qualifier, deploymentIdent, label)
	} else {
		base = fmt.Sprintf("%s-%s", qualifier, label)
	}

	return applyCasing(base, casing)
}

func applyCasing(s string, casing Casing) string {
	switch casing {
	case CasingCamel:
		return strcase.ToCamel(s)
	case CasingLowerCamel:
		return strcase.ToLowerCamel(s)
	case CasingSnake:
		return strcase.ToSnake(s)
	case CasingScreamingSnake:
		return strcase.ToScreamingSnake(s)
	case CasingKebab:
		return strcase.ToKebab(s)
	case CasingScreamingKebab:
		return strcase.ToScreamingKebab(s)
	default:
		return strcase.ToCamel(s)
	}
}
