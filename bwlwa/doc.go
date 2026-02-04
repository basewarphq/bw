// Package bwlwa provides a batteries-included framework for building HTTP services
// that run on AWS Lambda with Lambda Web Adapter (LWA).
//
// # Overview
//
// bwlwa handles the boilerplate of setting up an HTTP server optimized for Lambda:
// environment parsing, structured logging, OpenTelemetry tracing, AWS SDK clients,
// and graceful shutdown. A complete application can be created in a single call:
//
//	bwlwa.NewApp[Env](func(m *bwlwa.Mux, h *Handlers) {
//	    m.HandleFunc("GET /items", h.ListItems)
//	    m.HandleFunc("GET /items/{id}", h.GetItem, "get-item")
//	},
//	    bwlwa.WithAWSClient(dynamodb.NewFromConfig),
//	    bwlwa.WithFx(fx.Provide(NewHandlers)),
//	).Run()
//
// # Environment Configuration
//
// Define your environment by embedding [BaseEnvironment]:
//
//	type Env struct {
//	    bwlwa.BaseEnvironment
//	    MainTableName string `env:"MAIN_TABLE_NAME,required"`
//	}
//
// BaseEnvironment provides the following environment variables:
//
//	| Variable                      | Required | Default | Description                                      |
//	|-------------------------------|----------|---------|--------------------------------------------------|
//	| AWS_LWA_PORT                  | Yes      | -       | Port the HTTP server listens on                  |
//	| AWS_LWA_READINESS_CHECK_PATH  | Yes      | -       | Health check endpoint path for LWA readiness     |
//	| AWS_REGION                    | Yes      | -       | AWS region (set automatically by Lambda runtime) |
//	| BW_SERVICE_NAME               | Yes      | -       | Service name for logging and tracing             |
//	| BW_PRIMARY_REGION             | Yes      | -       | Primary deployment region (injected by CDK)      |
//	| BW_LOG_LEVEL                  | No       | info    | Log level (debug, info, warn, error)             |
//	| BW_OTEL_EXPORTER              | No       | stdout  | Trace exporter: "stdout" or "xrayudp"            |
//
// The AWS_LWA_* variables match the official Lambda Web Adapter configuration,
// so values you set for LWA are automatically picked up by bwlwa.
// AWS_REGION is set automatically by the Lambda runtime, while BW_PRIMARY_REGION
// is injected by the bwcdklwalambda CDK construct.
//
// # Context Functions
//
// All request context is accessed through typed functions:
//
//   - [Log] returns a trace-correlated zap logger
//   - [Span] returns the current OpenTelemetry span for custom instrumentation
//   - [Env] retrieves the typed environment configuration
//   - [AWS] retrieves a registered AWS SDK client by type
//   - [LWA] retrieves Lambda execution context (request ID, deadline, etc.)
//   - [Reverse] generates URLs for named routes
//
// Example handler using context functions:
//
//	func (h *Handlers) GetItem(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
//	    log := bwlwa.Log(ctx)           // trace-correlated logger
//	    env := bwlwa.Env[Env](ctx)      // typed environment
//	    dynamo := bwlwa.AWS[dynamodb.Client](ctx)  // AWS client
//
//	    bwlwa.Span(ctx).AddEvent("fetching item")
//
//	    // ... handler logic
//	}
//
// # Tracing
//
// OpenTelemetry tracing is configured automatically based on OTEL_EXPORTER:
//
//   - "stdout" (default): Pretty-printed spans for local development
//   - "xrayudp": X-Ray UDP exporter for Lambda with proper trace ID format
//
// The tracer provider and propagator are injected explicitly (no globals),
// allowing for proper testing and isolation.
//
// # AWS Clients
//
// Register AWS SDK v2 clients with [WithAWSClient]:
//
//	bwlwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client {
//	    return dynamodb.NewFromConfig(cfg)
//	})
//
// Clients are automatically instrumented with OpenTelemetry and accessible
// via [AWS] in handlers.
//
// # Cross-Region AWS Clients
//
// By default, AWS clients target the local region (AWS_REGION). For cross-region
// operations, register clients for specific regions:
//
//	// Local region (default)
//	bwlwa.WithAWSClient(dynamodb.NewFromConfig)
//
//	// Primary deployment region (uses PRIMARY_REGION env var)
//	bwlwa.WithAWSClient(s3.NewFromConfig, bwlwa.ForPrimaryRegion())
//
//	// Fixed region
//	bwlwa.WithAWSClient(sqs.NewFromConfig, bwlwa.ForRegion("us-east-1"))
//
// Retrieve clients in handlers with an optional region argument:
//
//	dynamo := bwlwa.AWS[dynamodb.Client](ctx)                          // local region
//	s3Client := bwlwa.AWS[s3.Client](ctx, bwlwa.PrimaryRegion())       // primary region
//	sqsClient := bwlwa.AWS[sqs.Client](ctx, bwlwa.FixedRegion("us-east-1"))
//
// Common use cases for cross-region clients:
//   - Reading shared configuration from primary region DynamoDB/SSM
//   - Publishing to centralized SQS queues or SNS topics
//   - Accessing S3 buckets in specific regions
//
// # Health Checks
//
// A health endpoint is automatically registered at AWS_LWA_READINESS_CHECK_PATH
// (required env var). Lambda Web Adapter uses this to determine readiness.
// Customize with [WithHealthHandler].
//
// # Dependency Injection
//
// bwlwa uses [go.uber.org/fx] for dependency injection. Add custom providers
// with [WithFx]:
//
//	bwlwa.WithFx(
//	    fx.Provide(NewHandlers),
//	    fx.Provide(NewRepository),
//	)
package bwlwa
