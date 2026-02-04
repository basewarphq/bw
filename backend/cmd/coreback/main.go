package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/advdv/bhttp"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/basewarphq/bwapp/bwlwa"
	"go.uber.org/zap"
)

type Env struct {
	bwlwa.BaseEnvironment
	MainTableName string `env:"MAIN_TABLE_NAME"`
}

func main() {
	bwlwa.NewApp[Env](
		routing,
		bwlwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client {
			return dynamodb.NewFromConfig(cfg)
		}),
	).Run()
}

func routing(m *bwlwa.Mux) {
	m.HandleFunc("POST /l/authorize", handleAuthorize)
	m.HandleFunc("GET /g/{path...}", handleGateway)
	m.HandleFunc("/{path...}", handleCatchAll)
}

func handleCatchAll(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	body, _ := io.ReadAll(r.Body)
	log := bwlwa.Log(ctx)
	log.Info("catch-all",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Any("headers", r.Header),
		zap.String("body", string(body)),
	)
	w.WriteHeader(http.StatusNotFound)
	_, err := w.Write([]byte("not found: " + r.URL.Path))
	return err
}

func handleGateway(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := bwlwa.Log(ctx)
	env := bwlwa.Env[Env](ctx)
	dynamoClient := bwlwa.AWS[dynamodb.Client](ctx)

	log.Info("gateway",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Any("headers", r.Header),
	)

	path := strings.TrimPrefix(r.URL.Path, "/g")

	if env.MainTableName != "" {
		var limit int32 = 1
		out, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
			TableName: &env.MainTableName,
			Limit:     &limit,
		})
		if err != nil {
			log.Error("dynamodb error", zap.Error(err))
		} else {
			log.Info("table scan",
				zap.String("table", env.MainTableName),
				zap.Int32("count", out.Count),
			)
		}
	}

	_, err := w.Write([]byte("hello, world: " + path))
	return err
}

func handleAuthorize(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := bwlwa.Log(ctx)
	span := bwlwa.Span(ctx)

	span.AddEvent("authorize-start")

	log.Info("authorize",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("error reading body", zap.Error(err))
		http.Error(w, "error reading body", http.StatusBadRequest)
		return nil
	}
	log.Debug("authorize body", zap.String("body", string(body)))

	// LWA pass-through POSTs the raw TOKEN authorizer event as the request body.
	var req events.APIGatewayCustomAuthorizerRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Error("error decoding JSON", zap.Error(err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return nil
	}
	log.Info("parsed request",
		zap.String("type", req.Type),
		zap.String("methodArn", req.MethodArn),
		zap.String("token", req.AuthorizationToken),
	)

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
	log.Debug("authorize response", zap.String("response", string(respBytes)))
	_, err = w.Write(respBytes)
	return err
}
