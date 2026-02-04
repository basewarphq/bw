package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/advdv/bhttp"
	"github.com/advdv/bhttp/blwa"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Env struct {
	blwa.BaseEnvironment
	MainTableName  string `env:"MAIN_TABLE_NAME"`
	MainSecretName string `env:"MAIN_SECRET_NAME"`
}

func main() {
	blwa.NewApp[Env](
		routing,
		blwa.WithAWSClient(func(cfg aws.Config) *dynamodb.Client {
			return dynamodb.NewFromConfig(cfg)
		}),
		blwa.WithFx(fx.Provide(NewHandlers)),
	).Run()
}

type Handlers struct {
	rt     *blwa.Runtime[Env]
	dynamo *dynamodb.Client
}

func NewHandlers(rt *blwa.Runtime[Env], dynamo *dynamodb.Client) *Handlers {
	return &Handlers{rt: rt, dynamo: dynamo}
}

func routing(m *blwa.Mux, h *Handlers) {
	m.HandleFunc("POST /l/authorize", h.handleAuthorize)
	m.HandleFunc("GET /g/{path...}", h.handleGateway)
	m.HandleFunc("/{path...}", handleCatchAll)
}

func handleCatchAll(ctx *blwa.Context, w bhttp.ResponseWriter, r *http.Request) error {
	body, _ := io.ReadAll(r.Body)
	log := ctx.Log()
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

func (h *Handlers) handleGateway(ctx *blwa.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := ctx.Log()
	env := h.rt.Env()

	log.Info("gateway",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Any("headers", r.Header),
	)

	path := strings.TrimPrefix(r.URL.Path, "/g")

	if env.MainTableName != "" {
		var limit int32 = 1
		out, err := h.dynamo.Scan(ctx, &dynamodb.ScanInput{
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

	placeholder := "(secret not available)"
	if env.MainSecretName != "" {
		val, err := h.rt.Secret(ctx, env.MainSecretName, "placeholder")
		if err != nil {
			log.Warn("failed to fetch secret", zap.Error(err))
		} else {
			placeholder = val
		}
	}

	_, err := w.Write([]byte("hello, world: " + path + " | secret: " + placeholder))
	return err
}

func (h *Handlers) handleAuthorize(ctx *blwa.Context, w bhttp.ResponseWriter, r *http.Request) error {
	log := ctx.Log()
	span := ctx.Span()

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
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Error("error encoding response", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil
	}
	log.Debug("authorize response", zap.String("response", string(respBytes)))
	_, err = w.Write(respBytes)
	return err
}
