package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/advdv/bhttp"
	"github.com/basewarphq/bwapp/bwlwa"
)

type Env struct {
	bwlwa.BaseEnvironment
}

func main() {
	bwlwa.NewApp[Env](routing).Run()
}

func routing(m *bwlwa.Mux) {
	m.HandleFunc("GET /", handleRoot)
	m.HandleFunc("GET /c/{path...}", handleConsole)
}

func handleRoot(_ context.Context, w bhttp.ResponseWriter, _ *http.Request) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Basewarp Console</title></head>
<body>
<h1>Basewarp Console</h1>
<p>Welcome to the Basewarp Console.</p>
</body>
</html>`))
	return err
}

func handleConsole(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
	path := strings.TrimPrefix(r.URL.Path, "/c")
	bwlwa.Log(ctx).Info("console request", )

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write([]byte("<!DOCTYPE html><html><head><title>Console</title></head><body><h1>Console</h1><p>Path: " + path + "</p></body></html>"))
	return err
}
