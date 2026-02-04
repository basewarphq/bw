package bwlwa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func TestNewExporter(t *testing.T) {
	ctx := context.Background()

	t.Run("stdout exporter", func(t *testing.T) {
		exp, err := newExporter(ctx, "stdout")
		if err != nil {
			t.Fatalf("newExporter(stdout) error: %v", err)
		}
		if exp == nil {
			t.Fatal("expected non-nil exporter")
		}
	})

	t.Run("empty defaults to stdout", func(t *testing.T) {
		exp, err := newExporter(ctx, "")
		if err != nil {
			t.Fatalf("newExporter('') error: %v", err)
		}
		if exp == nil {
			t.Fatal("expected non-nil exporter")
		}
	})

	t.Run("unsupported exporter returns error", func(t *testing.T) {
		_, err := newExporter(ctx, "invalid")
		if err == nil {
			t.Fatal("expected error for unsupported exporter")
		}
		if got := err.Error(); got != `unsupported OTEL_EXPORTER: "invalid" (supported: stdout, xrayudp)` {
			t.Errorf("unexpected error message: %s", got)
		}
	})
}

func TestNewResource(t *testing.T) {
	ctx := context.Background()

	t.Run("stdout resource has service name", func(t *testing.T) {
		res, err := newResource(ctx, "stdout", "my-service")
		if err != nil {
			t.Fatalf("newResource error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil resource")
		}

		found := false
		for _, attr := range res.Attributes() {
			if string(attr.Key) == "service.name" && attr.Value.AsString() == "my-service" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected service.name attribute in resource")
		}
	})

	t.Run("empty exporter type uses stdout resource", func(t *testing.T) {
		res, err := newResource(ctx, "", "test-service")
		if err != nil {
			t.Fatalf("newResource error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil resource")
		}
	})
}

func TestNewTracerProvider_Stdout(t *testing.T) {
	env := testEnv{otelExp: "stdout"}

	var tp trace.TracerProvider
	app := fx.New(
		fx.NopLogger,
		fx.Supply(fx.Annotate(env, fx.As(new(Environment)))),
		fx.Provide(NewTracerProvider),
		fx.Invoke(func(p trace.TracerProvider) { tp = p }),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("app.Start error: %v", err)
	}

	if tp == nil {
		t.Fatal("expected tracer provider to be set")
	}
	if _, ok := tp.(*sdktrace.TracerProvider); !ok {
		t.Error("expected SDK TracerProvider")
	}

	if err := app.Stop(ctx); err != nil {
		t.Fatalf("app.Stop error: %v", err)
	}
}

func TestNewPropagator(t *testing.T) {
	t.Run("stdout uses composite propagator", func(t *testing.T) {
		env := testEnv{otelExp: "stdout"}
		prop := NewPropagator(env)
		if prop == nil {
			t.Fatal("expected propagator to be set")
		}
		if _, ok := prop.(propagation.TraceContext); ok {
			t.Error("expected composite propagator, not just TraceContext")
		}
	})

	t.Run("empty defaults to composite propagator", func(t *testing.T) {
		env := testEnv{otelExp: ""}
		prop := NewPropagator(env)
		if prop == nil {
			t.Fatal("expected propagator to be set")
		}
	})
}

func TestNewTracerProvider_InvalidExporter(t *testing.T) {
	env := testEnv{otelExp: "invalid"}

	app := fx.New(
		fx.NopLogger,
		fx.Supply(fx.Annotate(env, fx.As(new(Environment)))),
		fx.Provide(NewTracerProvider),
		fx.Invoke(func(trace.TracerProvider) {}),
	)

	ctx := context.Background()
	err := app.Start(ctx)
	if err == nil {
		t.Fatal("expected error for invalid exporter")
		app.Stop(ctx)
	}
}

func TestNewTracerProvider_ShutdownHook(t *testing.T) {
	env := testEnv{otelExp: "stdout"}

	var hookCalled bool
	app := fx.New(
		fx.NopLogger,
		fx.Supply(fx.Annotate(env, fx.As(new(Environment)))),
		fx.Provide(NewTracerProvider),
		fx.Invoke(func(trace.TracerProvider) {}),
		fx.Invoke(func(lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					hookCalled = true
					return nil
				},
			})
		}),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("app.Start error: %v", err)
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatalf("app.Stop error: %v", err)
	}

	if !hookCalled {
		t.Error("expected shutdown hook to be called")
	}
}

func TestWithTracing(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tp := sdktrace.NewTracerProvider()
	prop := propagation.TraceContext{}

	t.Run("wraps handler with tracing", func(t *testing.T) {
		wrapped := withTracing(tp, prop, "test-service")(handler)
		if wrapped == nil {
			t.Fatal("expected non-nil handler")
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("excludes specified paths", func(t *testing.T) {
		wrapped := withTracing(tp, prop, "test-service", "/health", "/ready")(handler)

		for _, path := range []string{"/health", "/ready", "/api"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("path %s: expected 200, got %d", path, rec.Code)
			}
		}
	})
}
