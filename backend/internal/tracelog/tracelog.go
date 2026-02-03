// Package tracelog provides context-aware logging with X-Ray trace correlation.
//
// AWS recommends including trace_id and span_id in log messages for trace-log
// correlation. CloudWatch Logs Insights can then filter logs by trace ID, and
// X-Ray can display correlated logs in the trace timeline.
//
// Log format follows AWS conventions:
//
//	trace_id=<trace-id> span_id=<span-id> <message>
package tracelog

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel/trace"
)

// Printf logs a message with trace context extracted from ctx.
// If the context contains an active span, trace_id and span_id are prepended.
func Printf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(formatWithTrace(ctx, msg))
}

// Errorf logs an error with trace context and records it on the active span.
// Use this for errors that should appear in both logs and traces.
func Errorf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(formatWithTrace(ctx, msg))

	// Also record on span so error appears in X-Ray
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(fmt.Errorf(format, args...))
	}
}

func formatWithTrace(ctx context.Context, msg string) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return msg
	}

	sc := span.SpanContext()
	return fmt.Sprintf("trace_id=%s span_id=%s %s",
		sc.TraceID().String(),
		sc.SpanID().String(),
		msg,
	)
}
