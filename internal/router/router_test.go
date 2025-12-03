package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_Get(t *testing.T) {
	r := New()

	called := false
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRouter_MiddlewareOrder(t *testing.T) {
	var order []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "before1")
			next.ServeHTTP(w, r)
			order = append(order, "after1")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "before2")
			next.ServeHTTP(w, r)
			order = append(order, "after2")
		})
	}

	r := New(middleware1)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}, middleware2)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	expected := []string{"before1", "before2", "handler", "after2", "after1"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestRouter_Group(t *testing.T) {
	globalCalled := false
	groupCalled := false

	globalMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			globalCalled = true
			next.ServeHTTP(w, r)
		})
	}

	groupMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groupCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r := New(globalMiddleware)
	group := r.Group(groupMiddleware)

	group.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !globalCalled {
		t.Error("global middleware was not called")
	}
	if !groupCalled {
		t.Error("group middleware was not called")
	}
}
