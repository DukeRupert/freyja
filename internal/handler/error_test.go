package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukerupert/hiri/internal/domain"
)

func TestErrorCodeToHTTPStatus(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{domain.EINVALID, http.StatusBadRequest},
		{domain.EUNAUTHORIZED, http.StatusUnauthorized},
		{domain.EPAYMENT, http.StatusPaymentRequired},
		{domain.EFORBIDDEN, http.StatusForbidden},
		{domain.ENOTFOUND, http.StatusNotFound},
		{domain.ECONFLICT, http.StatusConflict},
		{domain.EGONE, http.StatusGone},
		{domain.ERATELIMIT, http.StatusTooManyRequests},
		{domain.EINTERNAL, http.StatusInternalServerError},
		{domain.ENOTIMPL, http.StatusNotImplemented},
		{"unknown_code", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := ErrorCodeToHTTPStatus(tt.code); got != tt.expected {
				t.Errorf("ErrorCodeToHTTPStatus(%q) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "not found error",
			err:            domain.NotFound("product.get", "product", "abc-123"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   domain.ENOTFOUND,
		},
		{
			name:           "validation error",
			err:            domain.Invalid("product.create", "price must be positive"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   domain.EINVALID,
		},
		{
			name:           "forbidden error",
			err:            domain.Forbidden("product.delete", "not authorized"),
			expectedStatus: http.StatusForbidden,
			expectedCode:   domain.EFORBIDDEN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept", "application/json")
			rec := httptest.NewRecorder()

			ErrorResponse(rec, req, tt.err)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			var response struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}

			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Error.Code != tt.expectedCode {
				t.Errorf("error.code = %q, want %q", response.Error.Code, tt.expectedCode)
			}
		})
	}
}

func TestErrorResponse_HTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	err := domain.NotFound("product.get", "product", "abc-123")
	ErrorResponse(rec, req, err)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Should be plain text, not JSON
	body := rec.Body.String()
	if body == "" {
		t.Error("response body should not be empty")
	}
}

func TestErrorResponse_InternalHidesDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	// Internal error with sensitive details
	err := domain.Internal(nil, "db.query", "failed to connect to database at 192.168.1.100:5432")
	ErrorResponse(rec, req, err)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should show generic message, not internal details
	expected := "An internal error occurred. Please try again later."
	if response.Error.Message != expected {
		t.Errorf("message = %q, want %q", response.Error.Message, expected)
	}
}

func TestValidationErrorResponse_JSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	err := domain.NewValidationError("product.create", "name", "name is required")
	err = domain.AddFieldError(err, "price", "price must be positive")

	ValidationErrorResponse(rec, req, err)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var response struct {
		Error struct {
			Code    string            `json:"code"`
			Message string            `json:"message"`
			Fields  map[string]string `json:"fields"`
		} `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error.Code != domain.EINVALID {
		t.Errorf("error.code = %q, want %q", response.Error.Code, domain.EINVALID)
	}

	if len(response.Error.Fields) != 2 {
		t.Errorf("fields count = %d, want 2", len(response.Error.Fields))
	}

	if response.Error.Fields["name"] != "name is required" {
		t.Errorf("fields[name] = %q, want %q", response.Error.Fields["name"], "name is required")
	}
}

func TestValidationErrorResponse_NonValidationError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	// Pass a non-validation error
	err := domain.NotFound("product.get", "product", "123")
	ValidationErrorResponse(rec, req, err)

	// Should fall back to regular error response
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestConvenienceResponses(t *testing.T) {
	t.Run("NotFoundResponse", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		NotFoundResponse(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("UnauthorizedResponse", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		UnauthorizedResponse(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("ForbiddenResponse", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		ForbiddenResponse(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("InternalErrorResponse", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		InternalErrorResponse(rec, req, nil)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestAcceptsJSON(t *testing.T) {
	tests := []struct {
		name        string
		accept      string
		contentType string
		path        string
		expected    bool
	}{
		{
			name:     "application/json in Accept",
			accept:   "application/json",
			expected: true,
		},
		{
			name:     "application/json with charset in Accept",
			accept:   "application/json; charset=utf-8",
			expected: true,
		},
		{
			name:        "application/json in Content-Type",
			contentType: "application/json",
			expected:    true,
		},
		{
			name:     ".json extension in path",
			path:     "/api/products.json",
			expected: true,
		},
		{
			name:   "text/html Accept",
			accept: "text/html",
			path:   "/products",
		},
		{
			name: "no headers",
			path: "/products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = "/test"
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			if got := acceptsJSON(req); got != tt.expected {
				t.Errorf("acceptsJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}
