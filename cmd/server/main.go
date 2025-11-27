package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"

	"github.com/dukerupert/freyja/internal/config"
)

type (
	Middleware func(http.Handler) http.Handler
	Router     struct {
		*http.ServeMux
		chain []Middleware
	}
)

func NewRouter(mx ...Middleware) *Router {
	return &Router{ServeMux: &http.ServeMux{}, chain: mx}
}

func (r *Router) Use(mx ...Middleware) {
	r.chain = append(r.chain, mx...)
}

func (r *Router) Group(fn func(r *Router)) {
	fn(&Router{ServeMux: r.ServeMux, chain: slices.Clone(r.chain)})
}

func (r *Router) Get(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodGet, path, fn, mx)
}

func (r *Router) Post(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodPost, path, fn, mx)
}

func (r *Router) Put(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodPut, path, fn, mx)
}

func (r *Router) Delete(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodDelete, path, fn, mx)
}

func (r *Router) Head(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodHead, path, fn, mx)
}

func (r *Router) Options(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodOptions, path, fn, mx)
}

func (r *Router) handle(method, path string, fn http.HandlerFunc, mx []Middleware) {
	r.Handle(method+" "+path, r.wrap(fn, mx))
}

func (r *Router) wrap(fn http.HandlerFunc, mx []Middleware) (out http.Handler) {
	out, mx = http.Handler(fn), append(slices.Clone(r.chain), mx...)

	slices.Reverse(mx)

	for _, m := range mx {
		out = m(out)
	}

	return
}

func mid(i int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("mid", i, "start")
			next.ServeHTTP(w, r)
			fmt.Println("mid", i, "done")
		})
	}
}

func someHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[the handler ran here]")
	fmt.Fprintln(w, "Hello world of", r.URL.Path)
}

func run() {
	// Load configuration
	_, err := config.New()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	r := NewRouter(mid(0))

	r.Group(func(r *Router) {
		r.Use(mid(1), mid(2))

		r.Get("/foo", someHandler)
	})

	r.Group(func(r *Router) {
		r.Use(mid(3))

		r.Get("/bar", someHandler, mid(4))
		r.Get("/baz", someHandler, mid(5))
	})

	r.Post("/foobar", someHandler)

	log.Fatal(http.ListenAndServe(":3000", r))
}