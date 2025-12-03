package storefront

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/google/uuid"
)

const (
	sessionCookieName = "freyja_session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
)

// SignupHandler handles the signup page and form submission
type SignupHandler struct {
	userService         service.UserService
	verificationService service.EmailVerificationService
	renderer            *handler.Renderer
	tenantID            uuid.UUID
}

// NewSignupHandler creates a new signup handler
func NewSignupHandler(
	userService service.UserService,
	verificationService service.EmailVerificationService,
	renderer *handler.Renderer,
	tenantID uuid.UUID,
) *SignupHandler {
	return &SignupHandler{
		userService:         userService,
		verificationService: verificationService,
		renderer:            renderer,
		tenantID:            tenantID,
	}
}

// ShowForm handles GET /signup - displays the signup form
func (h *SignupHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	h.showFormWithError(w, r, nil)
}

func (h *SignupHandler) showFormWithError(w http.ResponseWriter, r *http.Request, formError *string) {
	data := BaseTemplateData(r)

	if formError != nil {
		data["Error"] = *formError
	}

	h.renderer.RenderHTTP(w, "signup", data)
}

// HandleSubmit handles POST /signup - processes the signup form
func (h *SignupHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.GetLogger(ctx)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		logger.Error("signup: failed to parse form", "error", err)
		errMsg := "Invalid form data"
		h.showFormWithError(w, r, &errMsg)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")

	// Validate required fields
	if email == "" || password == "" {
		logger.Warn("signup: missing required fields", "email", email, "hasPassword", password != "")
		errMsg := "Email and password are required"
		h.showFormWithError(w, r, &errMsg)
		return
	}

	// Register user
	user, err := h.userService.Register(ctx, email, password, firstName, lastName)
	if err != nil {
		logger.Error("signup: registration failed",
			"email", email,
			"error", err,
			"isUserExists", errors.Is(err, service.ErrUserExists))
		var errMsg string
		if errors.Is(err, service.ErrUserExists) {
			errMsg = "An account with this email already exists"
		} else {
			errMsg = "Failed to create account. Please try again."
		}
		h.showFormWithError(w, r, &errMsg)
		return
	}

	logger.Info("signup: user registered successfully", "email", email, "userID", user.ID)

	// Convert user ID from pgtype.UUID to uuid.UUID
	userID, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		logger.Error("signup: failed to convert user ID", "error", err)
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	// Get IP and user agent for rate limiting
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	// Send verification email
	err = h.verificationService.SendVerificationEmail(ctx, h.tenantID, userID, email, firstName, ipAddress, userAgent)
	if err != nil {
		logger.Error("signup: failed to send verification email", "error", err)
		// Still show success - the user was created, just the email failed
	}

	logger.Info("signup: verification email sent", "email", email)

	// Redirect to verification pending page
	http.Redirect(w, r, "/signup-success?email="+email, http.StatusSeeOther)
}

// LoginHandler handles the login page and form submission
type LoginHandler struct {
	userService service.UserService
	renderer    *handler.Renderer
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(userService service.UserService, renderer *handler.Renderer) *LoginHandler {
	return &LoginHandler{
		userService: userService,
		renderer:    renderer,
	}
}

// ShowForm handles GET /login - displays the login form
func (h *LoginHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	h.showFormWithError(w, r, nil, "")
}

func (h *LoginHandler) showFormWithError(w http.ResponseWriter, r *http.Request, formError *string, email string) {
	data := BaseTemplateData(r)

	if formError != nil {
		data["Error"] = *formError
	}
	if email != "" {
		data["Email"] = email
	}

	// Check for password reset success message
	if r.URL.Query().Get("reset") == "success" {
		data["Success"] = "Your password has been reset successfully. Please log in with your new password."
	}

	h.renderer.RenderHTTP(w, "login", data)
}

// HandleSubmit handles POST /login - processes the login form
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

	// Authenticate user
	user, err := h.userService.Authenticate(ctx, email, password)
	if err != nil {
		if errors.Is(err, service.ErrEmailNotVerified) {
			// Redirect to resend verification page
			http.Redirect(w, r, "/resend-verification?email="+email, http.StatusSeeOther)
			return
		}
		var errMsg string
		if errors.Is(err, service.ErrInvalidPassword) || errors.Is(err, service.ErrUserNotFound) {
			errMsg = "Invalid email or password"
		} else if errors.Is(err, service.ErrAccountSuspended) {
			errMsg = "Your account has been suspended"
		} else if errors.Is(err, service.ErrAccountPending) {
			errMsg = "Your account is pending approval"
		} else {
			errMsg = "Login failed. Please try again."
		}
		h.showFormWithError(w, r, &errMsg, email)
		return
	}

	// Create session
	userIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		user.ID.Bytes[0:4], user.ID.Bytes[4:6], user.ID.Bytes[6:8],
		user.ID.Bytes[8:10], user.ID.Bytes[10:16])
	token, err := h.userService.CreateSession(ctx, userIDStr)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

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

	// Redirect to home page or returnTo URL
	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

// LogoutHandler handles user logout
type LogoutHandler struct {
	userService service.UserService
}

// NewLogoutHandler creates a new logout handler
func NewLogoutHandler(userService service.UserService) *LogoutHandler {
	return &LogoutHandler{
		userService: userService,
	}
}

// HandleSubmit handles POST /logout - logs out the user
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

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// SignupSuccessHandler handles the signup success page
type SignupSuccessHandler struct {
	renderer *handler.Renderer
}

// NewSignupSuccessHandler creates a new signup success handler
func NewSignupSuccessHandler(renderer *handler.Renderer) *SignupSuccessHandler {
	return &SignupSuccessHandler{
		renderer: renderer,
	}
}

// ServeHTTP handles GET /signup-success - displays the verification pending page
func (h *SignupSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(r)

	// Get email from query params
	email := r.URL.Query().Get("email")
	if email != "" {
		data["Email"] = email
	}

	h.renderer.RenderHTTP(w, "signup_success", data)
}
