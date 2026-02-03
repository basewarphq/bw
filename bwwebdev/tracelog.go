package bwwebdev

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel/trace"
)

// LogPrintf logs a message with trace context extracted from ctx.
// If the context contains an active span, trace_id and span_id are prepended.
func LogPrintf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(formatWithTrace(ctx, msg))
}

// LogErrorf logs an error with trace context and records it on the active span.
// Use this for errors that should appear in both logs and traces.
func LogErrorf(ctx context.Context, format string, args ...any) {
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
