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

// Primary wraps an AWS client for the primary deployment region.
// Use this when registering and injecting clients that must target PRIMARY_REGION.
//
// Registration:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.Primary[ssm.Client] {
//	    return bwlwa.NewPrimary(ssm.NewFromConfig(cfg))
//	}, bwlwa.ForPrimaryRegion())
//
// Injection:
//
//	func NewHandlers(ssm *bwlwa.Primary[ssm.Client]) *Handlers
//
// Usage:
//
//	h.ssm.Client.GetParameter(ctx, ...)
type Primary[T any] struct {
	Client *T
}

// newPrimary creates a Primary wrapper for an AWS client.
func newPrimary[T any](client *T) *Primary[T] {
	return &Primary[T]{Client: client}
}

// InRegion wraps an AWS client configured for a specific fixed region.
// Use this when registering and injecting clients that must target a specific region.
//
// Registration:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.InRegion[sqs.Client] {
//	    return bwlwa.NewInRegion(sqs.NewFromConfig(cfg), "us-east-1")
//	}, bwlwa.ForRegion("us-east-1"))
//
// Injection:
//
//	func NewHandlers(sqs *bwlwa.InRegion[sqs.Client]) *Handlers
//
// Usage:
//
//	h.sqs.Client.SendMessage(ctx, ...)
//	region := h.sqs.Region // "us-east-1"
type InRegion[T any] struct {
	Client *T
	Region string
}

// newInRegion creates an InRegion wrapper for an AWS client.
func newInRegion[T any](client *T, region string) *InRegion[T] {
	return &InRegion[T]{Client: client, Region: region}
}

// clientOptions holds configuration for AWS client registration.
type clientOptions struct {
	region Region
}

// ClientOption configures AWS client registration.
type ClientOption func(*clientOptions)

// ForPrimaryRegion configures the client to use the PRIMARY_REGION env var.
// Use this for cross-region operations that must target the primary deployment region.
//
// The factory should return *bwlwa.Primary[T] to make the region explicit in the type:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.Primary[ssm.Client] {
//	    return bwlwa.NewPrimary(ssm.NewFromConfig(cfg))
//	}, bwlwa.ForPrimaryRegion())
func ForPrimaryRegion() ClientOption {
	return func(o *clientOptions) {
		o.region = PrimaryRegion()
	}
}

// ForRegion configures the client to use a specific fixed region.
//
// The factory should return *bwlwa.InRegion[T] to make the region explicit in the type:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.InRegion[sqs.Client] {
//	    return bwlwa.NewInRegion(sqs.NewFromConfig(cfg), "us-east-1")
//	}, bwlwa.ForRegion("us-east-1"))
func ForRegion(region string) ClientOption {
	return func(o *clientOptions) {
		o.region = FixedRegion(region)
	}
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

// AWSClientProvider creates an fx.Option that provides an AWS client for injection.
// The factory receives an aws.Config with the region already configured.
//
// For local region clients (default), the factory returns *T directly:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client {
//	    return dynamodb.NewFromConfig(cfg)
//	})
//
// For primary region clients, wrap with Primary[T]:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.Primary[ssm.Client] {
//	    return bwlwa.NewPrimary(ssm.NewFromConfig(cfg))
//	}, bwlwa.ForPrimaryRegion())
//
// For fixed region clients, wrap with InRegion[T]:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.InRegion[sqs.Client] {
//	    return bwlwa.NewInRegion(sqs.NewFromConfig(cfg), "us-east-1")
//	}, bwlwa.ForRegion("us-east-1"))
func AWSClientProvider[T any](factory func(aws.Config) T, opts ...ClientOption) fx.Option {
	options := &clientOptions{
		region: LocalRegion(),
	}
	for _, opt := range opts {
		opt(options)
	}

	return fx.Provide(func(cfg aws.Config, env Environment) T {
		awsCfg := cfg.Copy()
		if options.region != nil {
			r := options.region.resolve(env)
			if r != "" {
				awsCfg.Region = r
			}
		}
		return factory(awsCfg)
	})
}

// NewPrimary creates a Primary wrapper for an AWS client configured for the primary region.
// Use this in your client factory when registering with ForPrimaryRegion():
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.Primary[ssm.Client] {
//	    return bwlwa.NewPrimary(ssm.NewFromConfig(cfg))
//	}, bwlwa.ForPrimaryRegion())
func NewPrimary[T any](client *T) *Primary[T] {
	return newPrimary(client)
}

// NewInRegion creates an InRegion wrapper for an AWS client configured for a fixed region.
// Use this in your client factory when registering with ForRegion():
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *bwlwa.InRegion[sqs.Client] {
//	    return bwlwa.NewInRegion(sqs.NewFromConfig(cfg), "us-east-1")
//	}, bwlwa.ForRegion("us-east-1"))
func NewInRegion[T any](client *T, region string) *InRegion[T] {
	return newInRegion(client, region)
}
