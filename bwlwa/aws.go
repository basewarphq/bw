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
	Factory func(aws.Config) any
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
func RegisterAWSClient[T any](factory func(aws.Config) *T) AWSClientFactory {
	return AWSClientFactory{
		TypeKey: typeKey[T](),
		Factory: func(cfg aws.Config) any { return factory(cfg) },
	}
}
