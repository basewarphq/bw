// Package tracing provides OpenTelemetry initialization for Lambda functions
// using AWS Lambda Web Adapter (LWA).
//
// The exporter is configurable via OTEL_EXPORTER environment variable:
//   - "xrayudp": Export directly to Lambda's built-in X-Ray daemon (no collector layer needed)
//   - "stdout": Print traces to stdout (for local development)
package tracing

import (
	"context"
	"fmt"
	"os"

	"github.com/aws-observability/aws-otel-go/exporters/xrayudp"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"go.opentelemetry.io/otel/sdk/trace"

)

var tp *trace.TracerProvider

// Init initializes OpenTelemetry with a configurable exporter.
// Call Shutdown before the function exits to flush pending traces.
//
// Set OTEL_EXPORTER to choose the exporter:
//   - "xrayudp": Export to Lambda's X-Ray daemon via UDP (default, no collector overhead)
//   - "stdout": Print traces to stdout (for local development)
func Init(ctx context.Context) error {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		return nil
	}

	exporter, err := newExporter(ctx)
	if err != nil {
		return err
	}

	// Detect Lambda resource attributes (function name, version, etc.).
	lambdaDetector := lambda.NewResourceDetector()
	res, err := lambdaDetector.Detect(ctx)
	if err != nil {
		return err
	}

	// Use synchronous span processor for Lambda.
	// With LWA, the HTTP server stays running but Lambda may freeze the container
	// between invocations. Sync export ensures spans are sent immediately,
	// avoiding data loss from unflushed batches.
	tp = trace.NewTracerProvider(
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
		trace.WithResource(res),
		trace.WithIDGenerator(xray.NewIDGenerator()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	return nil
}

func newExporter(ctx context.Context) (trace.SpanExporter, error) {
	switch os.Getenv("OTEL_EXPORTER") {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "xrayudp", "":
		// Default: export directly to Lambda's built-in X-Ray daemon via UDP.
		// No collector layer needed, eliminates ~20-25ms ADOT overhead.
		return xrayudp.NewSpanExporter(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER: %q (supported: xrayudp, stdout)", os.Getenv("OTEL_EXPORTER"))
	}
}

// Shutdown flushes pending traces and shuts down the tracer provider.
// Must be called before the Lambda function exits.
func Shutdown(ctx context.Context) error {
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}
