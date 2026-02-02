//nolint:paralleltest // jsii runtime doesn't support parallel tests
package bwcdkutil_test

import (
	"strings"
	"testing"

	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

func TestNewConfig(t *testing.T) {
	defer jsii.Close()

	tests := []struct {
		name        string
		context     map[string]any
		appConfig   bwcdkutil.AppConfig
		wantErr     bool
		errContains []string
	}{
		{
			name: "valid config",
			context: map[string]any{
				"myapp-qualifier":         "myapp",
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{"eu-west-1"},
				"myapp-deployments":       []any{"Dev", "Stag", "Prod"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr: false,
		},
		{
			name: "valid config without secondary regions",
			context: map[string]any{
				"myapp-qualifier":         "myapp",
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       []any{"Prod"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr: false,
		},
		{
			name: "missing qualifier",
			context: map[string]any{
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       []any{"Dev"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"myapp-qualifier", "is not set"},
		},
		{
			name: "qualifier too long",
			context: map[string]any{
				"myapp-qualifier":         "thisqualifieristoolong",
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       []any{"Prod"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"Qualifier", "exceeds maximum length"},
		},
		{
			name: "invalid base domain name",
			context: map[string]any{
				"myapp-qualifier":         "myapp",
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       []any{"Prod"},
				"myapp-base-domain-name":  "not a valid domain",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"BaseDomainName", "valid domain"},
		},
		{
			name: "unknown primary region",
			context: map[string]any{
				"myapp-qualifier":         "myapp",
				"myapp-primary-region":    "unknown-region-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       []any{"Dev"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"unknown primary region"},
		},
		{
			name: "unknown secondary region",
			context: map[string]any{
				"myapp-qualifier":         "myapp",
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{"eu-west-1", "unknown-region-2"},
				"myapp-deployments":       []any{"Dev"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"unknown secondary region"},
		},
		{
			name: "multiple errors",
			context: map[string]any{
				"myapp-secondary-regions": []any{},
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"myapp-qualifier", "myapp-primary-region", "myapp-deployments"},
		},
		{
			name: "wrong type for qualifier",
			context: map[string]any{
				"myapp-qualifier":         123, // should be string
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       []any{"Dev"},
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"myapp-qualifier", "must be a string"},
		},
		{
			name: "wrong type for deployments",
			context: map[string]any{
				"myapp-qualifier":         "myapp",
				"myapp-primary-region":    "us-east-1",
				"myapp-secondary-regions": []any{},
				"myapp-deployments":       "Dev", // should be array
				"myapp-base-domain-name":  "example.com",
			},
			appConfig: bwcdkutil.AppConfig{
				Prefix: "myapp-",
			},
			wantErr:     true,
			errContains: []string{"myapp-deployments", "must be an array"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := awscdk.NewApp(&awscdk.AppProps{
				Context: &tt.context,
			})

			cfg, err := bwcdkutil.NewConfig(app, tt.appConfig)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				for _, contains := range tt.errContains {
					if !strings.Contains(err.Error(), contains) {
						t.Errorf("error %q should contain %q", err.Error(), contains)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify config values
			if cfg.Qualifier != tt.context["myapp-qualifier"] {
				t.Errorf("Qualifier = %q, want %q", cfg.Qualifier, tt.context["myapp-qualifier"])
			}
			if cfg.PrimaryRegion != tt.context["myapp-primary-region"] {
				t.Errorf("PrimaryRegion = %q, want %q", cfg.PrimaryRegion, tt.context["myapp-primary-region"])
			}
		})
	}
}

func TestConfig_AllRegions(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(&awscdk.AppProps{
		Context: &map[string]any{
			"myapp-qualifier":         "myapp",
			"myapp-primary-region":    "us-east-1",
			"myapp-secondary-regions": []any{"eu-west-1", "ap-southeast-1"},
			"myapp-deployments":       []any{"Prod"},
			"myapp-base-domain-name":  "example.com",
		},
	})

	cfg, err := bwcdkutil.NewConfig(app, bwcdkutil.AppConfig{
		Prefix: "myapp-",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	regions := cfg.AllRegions()
	if len(regions) != 3 {
		t.Fatalf("AllRegions() = %v, want 3 regions", regions)
	}
	if regions[0] != "us-east-1" {
		t.Errorf("AllRegions()[0] = %q, want %q", regions[0], "us-east-1")
	}
	if regions[1] != "eu-west-1" {
		t.Errorf("AllRegions()[1] = %q, want %q", regions[1], "eu-west-1")
	}
	if regions[2] != "ap-southeast-1" {
		t.Errorf("AllRegions()[2] = %q, want %q", regions[2], "ap-southeast-1")
	}
}

func TestConfig_RegionIdent(t *testing.T) {
	defer jsii.Close()

	app := awscdk.NewApp(&awscdk.AppProps{
		Context: &map[string]any{
			"myapp-qualifier":         "myapp",
			"myapp-primary-region":    "us-east-1",
			"myapp-secondary-regions": []any{"eu-west-1"},
			"myapp-deployments":       []any{"Prod"},
			"myapp-base-domain-name":  "example.com",
		},
	})

	cfg, err := bwcdkutil.NewConfig(app, bwcdkutil.AppConfig{
		Prefix: "myapp-",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ident := cfg.RegionIdent("us-east-1"); ident != "Use1" {
		t.Errorf("RegionIdent(us-east-1) = %q, want %q", ident, "Use1")
	}
	if ident := cfg.RegionIdent("eu-west-1"); ident != "Euw1" {
		t.Errorf("RegionIdent(eu-west-1) = %q, want %q", ident, "Euw1")
	}
}
