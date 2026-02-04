package bwlwa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/advdv/bhttp"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/basewarphq/bwapp/bwlwa"
)

func TestAWSClient_DefaultTargetsLocalRegion(t *testing.T) {
	factory := bwlwa.RegisterAWSClient(func(cfg aws.Config) *dynamodb.Client {
		return dynamodb.NewFromConfig(cfg)
	})

	if factory.Region == nil {
		t.Fatal("expected Region to be set (LocalRegion by default)")
	}
}

func TestAWSClient_ForPrimaryRegion(t *testing.T) {
	factory := bwlwa.RegisterAWSClient(func(cfg aws.Config) *s3.Client {
		return s3.NewFromConfig(cfg)
	}, bwlwa.ForPrimaryRegion())

	if factory.Region == nil {
		t.Fatal("expected Region to be set")
	}
}

func TestAWSClient_ForFixedRegion(t *testing.T) {
	factory := bwlwa.RegisterAWSClient(func(cfg aws.Config) *sqs.Client {
		return sqs.NewFromConfig(cfg)
	}, bwlwa.ForRegion("ap-northeast-1"))

	if factory.Region == nil {
		t.Fatal("expected Region to be set")
	}
}

type regionTestEnv struct {
	bwlwa.BaseEnvironment
}

func TestAWS_RetrievesCorrectClient(t *testing.T) {
	t.Setenv("BW_PRIMARY_REGION", "eu-central-1")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_LWA_PORT", "18082")
	t.Setenv("BW_SERVICE_NAME", "region-test")
	t.Setenv("AWS_LWA_READINESS_CHECK_PATH", "/health")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	var localClient, primaryClient, fixedClient bool

	app := bwlwa.NewApp[regionTestEnv](
		func(m *bwlwa.Mux) {
			m.HandleFunc("GET /test", func(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
				// Should be able to retrieve the local region client (default)
				dynamo := bwlwa.AWS[dynamodb.Client](ctx)
				localClient = dynamo != nil

				// Should be able to retrieve primary region client with optional arg
				s3Client := bwlwa.AWS[s3.Client](ctx, bwlwa.PrimaryRegion())
				primaryClient = s3Client != nil

				// Should be able to retrieve fixed region client
				sqsClient := bwlwa.AWS[sqs.Client](ctx, bwlwa.FixedRegion("ap-northeast-1"))
				fixedClient = sqsClient != nil

				w.Header().Set("Content-Type", "application/json")
				return json.NewEncoder(w).Encode(map[string]bool{
					"local":   localClient,
					"primary": primaryClient,
					"fixed":   fixedClient,
				})
			})
		},
		bwlwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client { return dynamodb.NewFromConfig(cfg) }),
		bwlwa.WithAWSClient(func(cfg aws.Config) *s3.Client { return s3.NewFromConfig(cfg) }, bwlwa.ForPrimaryRegion()),
		bwlwa.WithAWSClient(func(cfg aws.Config) *sqs.Client { return sqs.NewFromConfig(cfg) }, bwlwa.ForRegion("ap-northeast-1")),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = app.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:18082/test")
	if err != nil {
		t.Fatalf("GET /test failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !result["local"] {
		t.Error("local region client should not be nil")
	}
	if !result["primary"] {
		t.Error("primary region client should not be nil")
	}
	if !result["fixed"] {
		t.Error("fixed region client should not be nil")
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestAWS_VerifiesRegionInConfig(t *testing.T) {
	t.Setenv("BW_PRIMARY_REGION", "eu-central-1")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_LWA_PORT", "18083")
	t.Setenv("BW_SERVICE_NAME", "region-verify-test")
	t.Setenv("AWS_LWA_READINESS_CHECK_PATH", "/health")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	var capturedLocalRegion, capturedPrimaryRegion, capturedFixedRegion string

	app := bwlwa.NewApp[regionTestEnv](
		func(m *bwlwa.Mux) {
			m.HandleFunc("GET /test", func(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				return nil
			})
		},
		bwlwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client {
			capturedLocalRegion = cfg.Region
			return dynamodb.NewFromConfig(cfg)
		}),
		bwlwa.WithAWSClient(func(cfg aws.Config) *s3.Client {
			capturedPrimaryRegion = cfg.Region
			return s3.NewFromConfig(cfg)
		}, bwlwa.ForPrimaryRegion()),
		bwlwa.WithAWSClient(func(cfg aws.Config) *sqs.Client {
			capturedFixedRegion = cfg.Region
			return sqs.NewFromConfig(cfg)
		}, bwlwa.ForRegion("ap-northeast-1")),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = app.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	if capturedLocalRegion != "eu-west-1" {
		t.Errorf("local client region = %q, want %q", capturedLocalRegion, "eu-west-1")
	}
	if capturedPrimaryRegion != "eu-central-1" {
		t.Errorf("primary client region = %q, want %q", capturedPrimaryRegion, "eu-central-1")
	}
	if capturedFixedRegion != "ap-northeast-1" {
		t.Errorf("fixed client region = %q, want %q", capturedFixedRegion, "ap-northeast-1")
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}
