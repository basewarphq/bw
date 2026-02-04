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
//	| SERVICE_NAME                  | Yes      | -       | Service name for logging and tracing             |
//	| AWS_LWA_READINESS_CHECK_PATH  | Yes      | -       | Health check endpoint path for LWA readiness     |
//	| LOG_LEVEL                     | No       | info    | Log level (debug, info, warn, error)             |
//	| OTEL_EXPORTER                 | No       | stdout  | Trace exporter: "stdout" or "xrayudp"            |
//
// The AWS_LWA_* variables match the official Lambda Web Adapter configuration,
// so values you set for LWA are automatically picked up by bwlwa.
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
