package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/advdv/bhttp"
	"github.com/advdv/bhttp/blwa"
)

type Env struct {
	blwa.BaseEnvironment
}

func main() {
	blwa.NewApp[Env](routing).Run()
}

func routing(m *blwa.Mux) {
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
	blwa.Log(ctx).Info("console request")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write([]byte(
		"<!DOCTYPE html><html><head><title>Console</title></head><body>" +
			"<h1>Console</h1><p>Path: " + path + "</p></body></html>"))
	return err
}
