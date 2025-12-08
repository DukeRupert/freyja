package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/middleware"
)

// ErrorResponse writes an error response to the client.
// It maps domain error codes to HTTP status codes and formats appropriately
// based on the Accept header (JSON for API requests, plain text otherwise).
func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	// Extract structured info from error
	code := domain.ErrorCode(err)
	message := domain.ErrorMessage(err)
	op := domain.ErrorOp(err)

	// Map to HTTP status
	status := ErrorCodeToHTTPStatus(code)

	// Log error with context
	logger := middleware.GetLogger(r.Context())
	logError(logger, r, err, code, op, status)

	// Check if request expects JSON
	if acceptsJSON(r) {
		writeJSONError(w, status, code, message)
		return
	}

	// Plain text error for HTML responses
	http.Error(w, message, status)
}

// ErrorCodeToHTTPStatus maps domain error codes to HTTP status codes.
func ErrorCodeToHTTPStatus(code string) int {
	switch code {
	case domain.EINVALID:
		return http.StatusBadRequest // 400
	case domain.EUNAUTHORIZED:
		return http.StatusUnauthorized // 401
	case domain.EPAYMENT:
		return http.StatusPaymentRequired // 402
	case domain.EFORBIDDEN:
		return http.StatusForbidden // 403
	case domain.ENOTFOUND:
		return http.StatusNotFound // 404
	case domain.ECONFLICT:
		return http.StatusConflict // 409
	case domain.EGONE:
		return http.StatusGone // 410
	case domain.ETOOLARGE:
		return http.StatusRequestEntityTooLarge // 413
	case domain.ERATELIMIT:
		return http.StatusTooManyRequests // 429
	case domain.EINTERNAL:
		return http.StatusInternalServerError // 500
	case domain.ENOTIMPL:
		return http.StatusNotImplemented // 501
	default:
		return http.StatusInternalServerError // 500
	}
}

// ValidationErrorResponse writes validation errors (field-level) to the response.
// For JSON requests, returns structured field errors.
// For HTML requests, returns a simple error message (forms should re-render with errors).
func ValidationErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		// Not a validation error, fall back to standard error response
		ErrorResponse(w, r, err)
		return
	}

	logger := middleware.GetLogger(r.Context())
	logger.Info("validation error",
		"op", ve.Op,
		"field_count", len(ve.Fields),
		"path", r.URL.Path,
	)

	if acceptsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    domain.EINVALID,
				"message": "Validation failed",
				"fields":  ve.Fields,
			},
		})
		return
	}

	// For HTML forms, return simple error message
	// Forms should typically re-render with errors inline
	http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
}

// NotFoundResponse is a convenience wrapper for 404 errors.
func NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.ENOTFOUND, "", "The requested resource was not found")
	ErrorResponse(w, r, err)
}

// UnauthorizedResponse is a convenience wrapper for 401 errors.
func UnauthorizedResponse(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.EUNAUTHORIZED, "", "Authentication required")
	ErrorResponse(w, r, err)
}

// ForbiddenResponse is a convenience wrapper for 403 errors.
func ForbiddenResponse(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.EFORBIDDEN, "", "You don't have permission to access this resource")
	ErrorResponse(w, r, err)
}

// InternalErrorResponse logs the error and returns a generic 500 response.
// The underlying error details are hidden from the user.
func InternalErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	wrappedErr := domain.Internal(err, "", "An unexpected error occurred")
	ErrorResponse(w, r, wrappedErr)
}

// logError logs the error with appropriate level based on status code.
func logError(logger *slog.Logger, r *http.Request, err error, code, op string, status int) {
	attrs := []any{
		"error", err.Error(),
		"code", code,
		"path", r.URL.Path,
		"method", r.Method,
		"status", status,
	}

	// Add operation if present
	if op != "" {
		attrs = append(attrs, "op", op)
	}

	// Add request ID if available
	if reqID := middleware.GetRequestID(r.Context()); reqID != "" {
		attrs = append(attrs, "request_id", reqID)
	}

	// Log level based on status code:
	// - 5xx errors are warnings/errors (server-side issues)
	// - 4xx errors are info (client errors, expected)
	if status >= 500 {
		logger.Error("server error", attrs...)
	} else if status >= 400 {
		logger.Info("client error", attrs...)
	}
}

// acceptsJSON checks if the client prefers JSON responses.
func acceptsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")

	// Check Accept header
	if strings.Contains(accept, "application/json") {
		return true
	}

	// Check if request body was JSON (API request)
	if strings.Contains(contentType, "application/json") {
		return true
	}

	// Check for .json extension in path (convenience for cURL)
	if strings.HasSuffix(r.URL.Path, ".json") {
		return true
	}

	return false
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// JSONError is a typed response structure for API errors.
// Can be used when you need to construct error responses manually.
type JSONError struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields,omitempty"`
	} `json:"error"`
}
