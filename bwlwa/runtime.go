package bwlwa

// Runtime provides access to app-scoped dependencies.
// Inject this into handler constructors via fx instead of pulling from context.
//
// Example:
//
//	type Handlers struct {
//	    rt     *bwlwa.Runtime[Env]
//	    dynamo *dynamodb.Client
//	}
//
//	func NewHandlers(rt *bwlwa.Runtime[Env], dynamo *dynamodb.Client) *Handlers {
//	    return &Handlers{rt: rt, dynamo: dynamo}
//	}
//
//	func (h *Handlers) GetItem(ctx context.Context, w bhttp.ResponseWriter, r *http.Request) error {
//	    env := h.rt.Env()
//	    url, _ := h.rt.Reverse("get-item", id)
//	    h.dynamo.GetItem(ctx, ...)
//	    // ...
//	}
type Runtime[E Environment] struct {
	env E
	mux *Mux
}

// NewRuntime creates a new Runtime with the given dependencies.
func NewRuntime[E Environment](env E, mux *Mux) *Runtime[E] {
	return &Runtime[E]{
		env: env,
		mux: mux,
	}
}

// Env returns the environment configuration.
func (r *Runtime[E]) Env() E {
	return r.env
}

// Reverse returns the URL for a named route with the given parameters.
// The route must have been registered with a name using Handle/HandleFunc.
func (r *Runtime[E]) Reverse(name string, params ...string) (string, error) {
	return r.mux.Reverse(name, params...)
}
