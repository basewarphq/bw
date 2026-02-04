package bwlwa_test

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/advdv/bhttp"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/basewarphq/bwapp/bwlwa"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Env defines the environment variables for the application.
// Embed bwlwa.BaseEnvironment to get the required LWA fields.
type Env struct {
	bwlwa.BaseEnvironment
	MainTableName string `env:"MAIN_TABLE_NAME,required"`
}

// ItemHandlers contains the HTTP handlers for item operations.
type ItemHandlers struct{}

func NewItemHandlers() *ItemHandlers { return &ItemHandlers{} }

// ListItems returns all items from the database.
// Demonstrates: Log for trace-correlated logging, Env for configuration access.
func (h *ItemHandlers) ListItems(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := bwlwa.Log(ctx)
	env := bwlwa.Env[Env](ctx)

	log.Info("listing items from table",
		zap.String("table", env.MainTableName))

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]any{
		"table": env.MainTableName,
		"items": []string{"item-1", "item-2"},
	})
}

// GetItem returns a single item by ID.
// Demonstrates: Span for adding trace events, Reverse for URL generation.
func (h *ItemHandlers) GetItem(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	span := bwlwa.Span(ctx)
	span.AddEvent("fetching item")

	selfURL, _ := bwlwa.Reverse(ctx, "get-item", id)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]any{
		"id":   id,
		"self": selfURL,
	})
}

// CreateItem creates a new item in DynamoDB.
// Demonstrates: AWS for typed client access, LWA for Lambda context.
func (h *ItemHandlers) CreateItem(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := bwlwa.Log(ctx)
	dynamo := bwlwa.AWS[dynamodb.Client](ctx)

	// Check if running in Lambda (LWA returns nil outside Lambda).
	if lwa := bwlwa.LWA(ctx); lwa != nil {
		log.Info("lambda context",
			zap.String("request_id", lwa.RequestID),
			zap.Duration("remaining", lwa.RemainingTime()),
		)
	}

	// Use the DynamoDB client (simplified example).
	_ = dynamo

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(map[string]string{
		"id":     "new-item-123",
		"status": "created",
	})
}

// GetConfig fetches configuration from the primary region SSM Parameter Store.
// Demonstrates: Cross-region AWS client access using PrimaryRegion().
func (h *ItemHandlers) GetConfig(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := bwlwa.Log(ctx)

	// Get SSM client for primary region - reads shared config across all regions.
	ssmClient := bwlwa.AWS[ssm.Client](ctx, bwlwa.PrimaryRegion())

	log.Info("fetching config from primary region SSM")

	// Use the SSM client (simplified example).
	_ = ssmClient

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{
		"config": "value-from-primary-region",
	})
}

// Example demonstrates a complete bwlwa application with four endpoints
// showcasing Log, Env, Span, Reverse, AWS, LWA, and cross-region client access.
func Example() {
	bwlwa.NewApp[Env](
		func(m *bwlwa.Mux, h *ItemHandlers) {
			m.HandleFunc("GET /items", h.ListItems)
			m.HandleFunc("GET /items/{id}", h.GetItem, "get-item")
			m.HandleFunc("POST /items", h.CreateItem)
			m.HandleFunc("GET /config", h.GetConfig)
		},
		// DynamoDB client for local region (default).
		bwlwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client {
			return dynamodb.NewFromConfig(cfg)
		}),
		// SSM client for primary region - reads shared config across all deployments.
		bwlwa.WithAWSClient(func(cfg aws.Config) *ssm.Client {
			return ssm.NewFromConfig(cfg)
		}, bwlwa.ForPrimaryRegion()),
		bwlwa.WithFx(fx.Provide(NewItemHandlers)),
	).Run()
}
