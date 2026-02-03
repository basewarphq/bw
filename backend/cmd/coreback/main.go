package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/basewarphq/bwapp/backend/internal/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	ctx := context.Background()

	if err := tracing.Init(ctx); err != nil {
		log.Printf("failed to initialize tracing: %v", err)
	}
	defer func() {
		if err := tracing.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown tracing: %v", err)
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/l/authorize", handleAuthorize)
	mux.HandleFunc("/g/", handleGateway)
	mux.HandleFunc("/", handleCatchAll)

	handler := otelhttp.NewHandler(mux, "coreback",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
		otelhttp.WithFilter(func(r *http.Request) bool {
			// Don't trace LWA readiness checks - they create orphan traces.
			return r.URL.Path != "/health"
		}),
	)
	http.ListenAndServe(":"+os.Getenv("PORT"), handler)
}

func handleCatchAll(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("[catch-all] method=%s path=%s", r.Method, r.URL.Path)
	log.Printf("[catch-all] headers=%v", r.Header)
	log.Printf("[catch-all] body=%s", string(body))
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found: " + r.URL.Path))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleGateway(w http.ResponseWriter, r *http.Request) {
	log.Printf("[gateway] method=%s path=%s", r.Method, r.URL.Path)
	log.Printf("[gateway] headers=%v", r.Header)
	path := strings.TrimPrefix(r.URL.Path, "/g")
	w.Write([]byte("hello, world: " + path))
}

func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Extract trace context from Lambda runtime environment.
	// LWA pass-through doesn't propagate trace headers, so we extract from env var.
	ctx := r.Context()
	if traceID := os.Getenv("_X_AMZN_TRACE_ID"); traceID != "" {
		carrier := propagation.HeaderCarrier{"X-Amzn-Trace-Id": []string{traceID}}
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	}

	tracer := otel.Tracer("coreback")
	ctx, span := tracer.Start(ctx, "authorize")
	defer span.End()

	log.Printf("[authorize] method=%s path=%s", r.Method, r.URL.Path)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[authorize] error reading body: %v", err)
		http.Error(w, "error reading body", http.StatusBadRequest)
		return
	}
	log.Printf("[authorize] body=%s", string(body))

	// LWA pass-through POSTs the raw TOKEN authorizer event as the request body.
	// TOKEN events only contain: type, authorizationToken, methodArn
	var req events.APIGatewayCustomAuthorizerRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[authorize] error decoding JSON: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	log.Printf("[authorize] parsed request: type=%s methodArn=%s token=%s", req.Type, req.MethodArn, req.AuthorizationToken)

	// TODO: Validate the authorization token
	_ = req.AuthorizationToken

	resp := events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: "user",
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   "Allow",
					Resource: []string{"*"},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	respBytes, _ := json.Marshal(resp)
	log.Printf("[authorize] response=%s", string(respBytes))
	w.Write(respBytes)
}
