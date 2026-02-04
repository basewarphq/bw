package bwlwa

import (
	"context"
	"net/http"

	"github.com/advdv/bhttp"
)

// Mux is an alias for bhttp.ServeMux with standard context.
type Mux = bhttp.ServeMux[context.Context]

// NewMux creates a new Mux with sensible defaults.
func NewMux() *Mux {
	logger := bhttp.NewStdLogger(nil)
	return bhttp.NewCustomServeMux(
		bhttp.StdContextInit,
		-1, // unlimited buffer
		logger,
		http.NewServeMux(),
		bhttp.NewReverser(),
	)
}
