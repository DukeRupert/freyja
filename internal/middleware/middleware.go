package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
)

// ============================================================================
// MIDDLEWARE ERROR RESPONSE HELPERS
// ============================================================================
//
// These helpers provide consistent error responses for middleware.
// They mirror the handler.ErrorResponse patterns but are self-contained
// to avoid circular imports (handler imports middleware for GetLogger, etc.)

// respondWithError writes an error response to the client.
// For JSON requests, returns structured JSON error.
// For other requests, returns plain text error.
func respondWithError(w http.ResponseWriter, r *http.Request, err error) {
	code := domain.ErrorCode(err)
	message := domain.ErrorMessage(err)
	status := errorCodeToHTTPStatus(code)

	// Log the error
	logger := GetLogger(r.Context())
	if logger == nil {
		logger = slog.Default()
	}

	attrs := []any{
		"error", err.Error(),
		"code", code,
		"path", r.URL.Path,
		"method", r.Method,
		"status", status,
	}

	if reqID := GetRequestID(r.Context()); reqID != "" {
		attrs = append(attrs, "request_id", reqID)
	}

	if status >= 500 {
		logger.Error("middleware error", attrs...)
	} else {
		logger.Info("middleware error", attrs...)
	}

	// Check if request expects JSON
	if acceptsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    code,
				"message": message,
			},
		})
		return
	}

	// Plain text for HTML responses
	http.Error(w, message, status)
}

// respondNotFound is a convenience wrapper for 404 errors.
func respondNotFound(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.ENOTFOUND, "", "The requested resource was not found")
	respondWithError(w, r, err)
}

// respondUnauthorized is a convenience wrapper for 401 errors.
func respondUnauthorized(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.EUNAUTHORIZED, "", "Authentication required")
	respondWithError(w, r, err)
}

// respondForbidden is a convenience wrapper for 403 errors.
func respondForbidden(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.EFORBIDDEN, "", "You don't have permission to access this resource")
	respondWithError(w, r, err)
}

// respondInternalError logs the error and returns a generic 500 response.
func respondInternalError(w http.ResponseWriter, r *http.Request, err error) {
	wrappedErr := domain.Internal(err, "", "An unexpected error occurred")
	respondWithError(w, r, wrappedErr)
}

// respondTooManyRequests is a convenience wrapper for 429 errors.
func respondTooManyRequests(w http.ResponseWriter, r *http.Request) {
	err := domain.Errorf(domain.ERATELIMIT, "", "Too many requests")
	respondWithError(w, r, err)
}

// respondBadRequest is a convenience wrapper for 400 errors.
func respondBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	err := domain.Errorf(domain.EINVALID, "", "%s", message)
	respondWithError(w, r, err)
}

// respondTooLarge is a convenience wrapper for 413 errors.
func respondTooLarge(w http.ResponseWriter, r *http.Request, message string) {
	err := domain.Errorf(domain.ETOOLARGE, "", "%s", message)
	respondWithError(w, r, err)
}

// errorCodeToHTTPStatus maps domain error codes to HTTP status codes.
func errorCodeToHTTPStatus(code string) int {
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

// acceptsJSON checks if the client prefers JSON responses.
func acceptsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(accept, "application/json") {
		return true
	}
	if strings.Contains(contentType, "application/json") {
		return true
	}
	if strings.HasSuffix(r.URL.Path, ".json") {
		return true
	}

	return false
}
