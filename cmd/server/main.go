package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"
)

type (
	middleware func(http.Handler) http.Handler
	router     struct {
		*http.ServeMux
		chain []middleware
	}
)

func NewRouter(mx ...middleware) *router {
	return &router{ServeMux: &http.ServeMux{}, chain: mx}
}

func (r *router) Use(mx ...middleware) {
	r.chain = append(r.chain, mx...)
}

func (r *router) Group(fn func(r *router)) {
	fn(&router{ServeMux: r.ServeMux, chain: slices.Clone(r.chain)})
}

func (r *router) Get(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodGet, path, fn, mx)
}

func (r *router) Post(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodPost, path, fn, mx)
}

func (r *router) Put(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodPut, path, fn, mx)
}

func (r *router) Delete(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodDelete, path, fn, mx)
}

func (r *router) Head(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodHead, path, fn, mx)
}

func (r *router) Options(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodOptions, path, fn, mx)
}

func (r *router) handle(method, path string, fn http.HandlerFunc, mx []middleware) {
	r.Handle(method+" "+path, r.wrap(fn, mx))
}

func (r *router) wrap(fn http.HandlerFunc, mx []middleware) (out http.Handler) {
	out, mx = http.Handler(fn), append(slices.Clone(r.chain), mx...)

	slices.Reverse(mx)

	for _, m := range mx {
		out = m(out)
	}

	return
}

func mid(i int) middleware {
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

func main() {
	r := NewRouter(mid(0))

	r.Group(func(r *router) {
		r.Use(mid(1), mid(2))

		r.Get("/foo", someHandler)
	})

	r.Group(func(r *router) {
		r.Use(mid(3))

		r.Get("/bar", someHandler, mid(4))
		r.Get("/baz", someHandler, mid(5))
	})

	r.Post("/foobar", someHandler)

	log.Fatal(http.ListenAndServe(":3000", r))
}