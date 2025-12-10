package router

import (
	"net/http"
	"slices"
	"strings"
)

// Router wraps http.ServeMux with middleware chaining
type Router struct {
	mux   *http.ServeMux
	chain []Middleware
}

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// New creates a new Router with optional global middleware
func New(middleware ...Middleware) *Router {
	return &Router{
		mux:   http.NewServeMux(),
		chain: middleware,
	}
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Get registers a GET route
func (r *Router) Get(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.handle(http.MethodGet, pattern, handler, middleware)
}

// Post registers a POST route
func (r *Router) Post(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.handle(http.MethodPost, pattern, handler, middleware)
}

// Put registers a PUT route
func (r *Router) Put(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.handle(http.MethodPut, pattern, handler, middleware)
}

// Delete registers a DELETE route
func (r *Router) Delete(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.handle(http.MethodDelete, pattern, handler, middleware)
}

// Patch registers a PATCH route
func (r *Router) Patch(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.handle(http.MethodPatch, pattern, handler, middleware)
}

// Handle registers a route with explicit method
func (r *Router) Handle(method, pattern string, handler http.Handler, middleware ...Middleware) {
	r.mux.Handle(method+" "+pattern, r.wrap(handler, middleware))
}

// handle is the internal route registration function
func (r *Router) handle(method, pattern string, handler http.HandlerFunc, middleware []Middleware) {
	r.Handle(method, pattern, handler, middleware...)
}

// wrap applies middleware to a handler in reverse order
func (r *Router) wrap(handler http.Handler, middleware []Middleware) http.Handler {
	// Combine global middleware chain with route-specific middleware
	combined := append(slices.Clone(r.chain), middleware...)

	// Apply middleware in reverse order so they execute in the order defined
	slices.Reverse(combined)

	result := handler
	for _, m := range combined {
		result = m(result)
	}

	return result
}

// Group creates a sub-router with additional middleware
func (r *Router) Group(middleware ...Middleware) *Router {
	return &Router{
		mux:   r.mux,
		chain: append(slices.Clone(r.chain), middleware...),
	}
}

// Static serves files from a directory under the given route prefix
func (r *Router) Static(prefix, dir string) {
	fileServer := http.FileServer(http.Dir(dir))

	// Ensure prefix doesn't end with slash for pattern matching
	cleanPrefix := strings.TrimSuffix(prefix, "/")

	// Strip the prefix before serving
	stripped := http.StripPrefix(cleanPrefix, fileServer)

	// Wrap with MIME type handler to ensure correct Content-Type headers
	// (some deployment environments lack proper MIME type databases)
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if ct := mimeTypeByExtension(req.URL.Path); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		stripped.ServeHTTP(w, req)
	})

	// Register with GET method and wildcard pattern
	r.mux.Handle("GET "+cleanPrefix+"/{file...}", r.wrap(handler, nil))
}

// mimeTypeByExtension returns the MIME type for common static file extensions.
func mimeTypeByExtension(path string) string {
	switch {
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".gif"):
		return "image/gif"
	case strings.HasSuffix(path, ".webp"):
		return "image/webp"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(path, ".woff"):
		return "font/woff"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(path, ".ttf"):
		return "font/ttf"
	case strings.HasSuffix(path, ".webmanifest"):
		return "application/manifest+json"
	default:
		return ""
	}
}
