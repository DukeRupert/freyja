package admin

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
)

const (
	sessionCookieName = "freyja_session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
)

// LoginHandler handles the admin login page and form submission
type LoginHandler struct {
	userService domain.UserService
	renderer    *handler.Renderer
}

// NewLoginHandler creates a new admin login handler
func NewLoginHandler(userService domain.UserService, renderer *handler.Renderer) *LoginHandler {
	return &LoginHandler{
		userService: userService,
		renderer:    renderer,
	}
}

// ShowForm handles GET /admin/login - displays the admin login form
func (h *LoginHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	h.showFormWithError(w, r, nil, "")
}

func (h *LoginHandler) showFormWithError(w http.ResponseWriter, r *http.Request, formError *string, email string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
	}

	// Add CSRF token for form protection
	if csrfToken := middleware.GetCSRFToken(r.Context()); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	if formError != nil {
		data["Error"] = *formError
	}
	if email != "" {
		data["Email"] = email
	}

	h.renderer.RenderHTTP(w, "admin/login", data)
}

// HandleSubmit handles POST /admin/login - processes the admin login form
func (h *LoginHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showFormWithError(w, r, &errMsg, "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate required fields
	if email == "" || password == "" {
		errMsg := "Email and password are required"
		h.showFormWithError(w, r, &errMsg, email)
		return
	}

	// Get client info for audit logging
	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	// Authenticate user
	user, err := h.userService.Authenticate(ctx, email, password)
	if err != nil {
		// Audit log: failed login attempt
		slog.Warn("admin: login failed",
			"email", email,
			"reason", err.Error(),
			"ip", ipAddress,
			"user_agent", userAgent,
		)

		var errMsg string
		errCode := domain.ErrorCode(err)
		switch errCode {
		case domain.EUNAUTHORIZED:
			errMsg = "Invalid email or password"
		case domain.EFORBIDDEN:
			// Could be suspended or email not verified
			if domain.ErrorMessage(err) == domain.ErrAccountSuspended.Error() {
				errMsg = "Your account has been suspended"
			} else if domain.ErrorMessage(err) == domain.ErrEmailNotVerified.Error() {
				errMsg = "Please verify your email before logging in"
			} else {
				errMsg = domain.ErrorMessage(err)
			}
		case domain.ENOTFOUND:
			errMsg = "Invalid email or password"
		default:
			errMsg = "Login failed. Please try again."
		}
		h.showFormWithError(w, r, &errMsg, email)
		return
	}

	// Verify user is an admin
	if user.AccountType != domain.UserAccountTypeAdmin {
		// Audit log: non-admin attempted admin login
		slog.Warn("admin: non-admin login attempt",
			"email", email,
			"account_type", user.AccountType,
			"ip", ipAddress,
			"user_agent", userAgent,
		)

		errMsg := "Access denied. Admin credentials required."
		h.showFormWithError(w, r, &errMsg, email)
		return
	}

	// Create session
	userIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		user.ID.Bytes[0:4], user.ID.Bytes[4:6], user.ID.Bytes[6:8],
		user.ID.Bytes[8:10], user.ID.Bytes[10:16])
	token, err := h.userService.CreateSession(ctx, userIDStr)
	if err != nil {
		slog.Error("admin: failed to create session",
			"email", email,
			"user_id", userIDStr,
			"error", err,
		)
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Audit log: successful admin login
	slog.Info("admin: login successful",
		"email", email,
		"user_id", userIDStr,
		"ip", ipAddress,
		"user_agent", userAgent,
	)

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to admin dashboard
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// LogoutHandler handles admin logout
type LogoutHandler struct {
	userService domain.UserService
}

// NewLogoutHandler creates a new admin logout handler
func NewLogoutHandler(userService domain.UserService) *LogoutHandler {
	return &LogoutHandler{
		userService: userService,
	}
}

// HandleSubmit handles POST /admin/logout - logs out the admin user
func (h *LogoutHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session cookie
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		// Delete session from database
		_ = h.userService.DeleteSession(ctx, cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to admin login page
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
