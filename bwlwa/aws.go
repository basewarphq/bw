package bwlwa

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// AWSClientFactory holds a factory function for creating AWS clients.
type AWSClientFactory struct {
	TypeKey string
	Region  Region
	Factory func(aws.Config) any
}

// clientOptions holds configuration for AWS client registration.
type clientOptions struct {
	region Region
}

// ClientOption configures AWS client registration.
type ClientOption func(*clientOptions)

// ForPrimaryRegion configures the client to use the PRIMARY_REGION env var.
// Use this for cross-region operations that must target the primary deployment region.
func ForPrimaryRegion() ClientOption {
	return func(o *clientOptions) {
		o.region = PrimaryRegion()
	}
}

// ForRegion configures the client to use a specific fixed region.
func ForRegion(region string) ClientOption {
	return func(o *clientOptions) {
		o.region = FixedRegion(region)
	}
}

// clientKey returns the storage key for a client type and region.
func clientKey(typeKey string, region Region, env Environment) string {
	if region == nil {
		return typeKey
	}
	r := region.resolve(env)
	if r == "" {
		return typeKey
	}
	return typeKey + "@" + r
}

const awsConfigTimeout = 10 * time.Second

// NewAWSConfig loads the default AWS SDK v2 configuration.
func NewAWSConfig(ctx context.Context) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(ctx)
}

// provideAWSConfig is an fx provider that loads AWS config with a timeout.
// It automatically instruments the config with OpenTelemetry for AWS SDK tracing.
// The TracerProvider and Propagator are explicitly injected to avoid global state.
func provideAWSConfig(lc fx.Lifecycle, tp trace.TracerProvider, prop propagation.TextMapPropagator) (aws.Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), awsConfigTimeout)
	defer cancel()
	cfg, err := NewAWSConfig(ctx)
	if err != nil {
		return cfg, err
	}
	otelaws.AppendMiddlewares(&cfg.APIOptions,
		otelaws.WithTracerProvider(tp),
		otelaws.WithTextMapPropagator(prop),
	)
	return cfg, nil
}

// RegisterAWSClient creates a factory for a typed AWS client.
// By default, clients target the local region (AWS_REGION env var).
// Use ForPrimaryRegion() to target the primary deployment region for cross-region operations.
// Use ForRegion("eu-west-1") to target a specific region.
func RegisterAWSClient[T any](factory func(aws.Config) *T, opts ...ClientOption) AWSClientFactory {
	options := &clientOptions{
		region: LocalRegion(), // default to local region
	}
	for _, opt := range opts {
		opt(options)
	}
	return AWSClientFactory{
		TypeKey: typeKey[T](),
		Region:  options.region,
		Factory: func(cfg aws.Config) any { return factory(cfg) },
	}
}
