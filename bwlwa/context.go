package bwlwa

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/advdv/bhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ctxKey is the key type for context values.
type ctxKey int

const (
	ctxKeyDeps ctxKey = iota
	ctxKeyLWAContext
)

// deps holds all dependencies available via context.
type deps struct {
	logger     *zap.Logger
	env        any
	mux        *Mux
	awsClients map[string]any
}

// LWAContext contains Lambda execution context from the x-amzn-lambda-context header.
type LWAContext struct {
	RequestID          string       `json:"request_id"`
	Deadline           int64        `json:"deadline"`
	InvokedFunctionARN string       `json:"invoked_function_arn"`
	XRayTraceID        string       `json:"xray_trace_id"`
	EnvConfig          LWAEnvConfig `json:"env_config"`
}

// LWAEnvConfig contains Lambda function environment configuration.
type LWAEnvConfig struct {
	FunctionName string `json:"function_name"`
	Memory       int    `json:"memory"`
	Version      string `json:"version"`
	LogGroup     string `json:"log_group"`
	LogStream    string `json:"log_stream"`
}

// DeadlineTime returns the Lambda invocation deadline as a time.Time.
func (lc *LWAContext) DeadlineTime() time.Time {
	if lc.Deadline == 0 {
		return time.Time{}
	}
	return time.UnixMilli(lc.Deadline)
}

// RemainingTime returns the duration until the Lambda invocation deadline.
func (lc *LWAContext) RemainingTime() time.Duration {
	if lc.Deadline == 0 {
		return 0
	}
	remaining := time.Until(lc.DeadlineTime())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// withDeps injects dependencies into the request context.
func withDeps(d *deps) bhttp.Middleware {
	return func(next bhttp.BareHandler) bhttp.BareHandler {
		return bhttp.BareHandlerFunc(func(w bhttp.ResponseWriter, r *http.Request) error {
			ctx := context.WithValue(r.Context(), ctxKeyDeps, d)
			return next.ServeBareBHTTP(w, r.WithContext(ctx))
		})
	}
}

// withLWAContext parses the x-amzn-lambda-context header from AWS Lambda Web Adapter.
func withLWAContext() bhttp.Middleware {
	return func(next bhttp.BareHandler) bhttp.BareHandler {
		return bhttp.BareHandlerFunc(func(w bhttp.ResponseWriter, r *http.Request) error {
			ctx := r.Context()
			if header := r.Header.Get("x-amzn-lambda-context"); header != "" {
				var lc LWAContext
				if err := json.Unmarshal([]byte(header), &lc); err == nil {
					ctx = context.WithValue(ctx, ctxKeyLWAContext, &lc)
				}
			}
			return next.ServeBareBHTTP(w, r.WithContext(ctx))
		})
	}
}

func depsFromContext(ctx context.Context) *deps {
	d, ok := ctx.Value(ctxKeyDeps).(*deps)
	if !ok {
		panic("bwlwa: deps not found in context; is the middleware configured?")
	}
	return d
}

// LWA retrieves the LWAContext from the request context.
// Returns nil if not running in a Lambda environment.
func LWA(ctx context.Context) *LWAContext {
	lc, _ := ctx.Value(ctxKeyLWAContext).(*LWAContext)
	return lc
}

// Log returns a trace-correlated zap logger from the context.
func Log(ctx context.Context) *zap.Logger {
	d := depsFromContext(ctx)
	return d.logger.With(traceFields(ctx)...)
}

// Span returns the current trace span from the context.
func Span(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// traceFields extracts trace_id and span_id from the context for log correlation.
func traceFields(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	sc := span.SpanContext()
	return []zap.Field{
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	}
}

// Env retrieves the environment configuration from the context.
func Env[E Environment](ctx context.Context) E {
	d := depsFromContext(ctx)
	env, ok := d.env.(E)
	if !ok {
		panic("bwlwa: environment type mismatch")
	}
	return env
}

// AWS retrieves a registered AWS client by type from context.
func AWS[T any](ctx context.Context) *T {
	d := depsFromContext(ctx)
	key := typeKey[T]()
	client, ok := d.awsClients[key]
	if !ok {
		panic("bwlwa: AWS client " + key + " not found; use WithAWSClient()")
	}
	return client.(*T)
}

// Reverse returns the URL for a named route with the given parameters.
// The route must have been registered with a name using Handle/HandleFunc.
func Reverse(ctx context.Context, name string, params ...string) (string, error) {
	d := depsFromContext(ctx)
	return d.mux.Reverse(name, params...)
}

// typeKey returns a unique string key for a type.
func typeKey[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		var ptr *T
		t = reflect.TypeOf(ptr).Elem()
	}
	return t.PkgPath() + "." + t.Name()
}
