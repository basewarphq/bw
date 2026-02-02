package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/l/authorize", handleAuthorize)
	mux.HandleFunc("/g/", handleGateway)
	mux.HandleFunc("/", handleCatchAll)

	http.ListenAndServe(":"+os.Getenv("PORT"), mux)
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
