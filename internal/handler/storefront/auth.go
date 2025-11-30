package storefront

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
)

const (
	sessionCookieName = "freyja_session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
)

// SignupHandler handles the signup page and form submission
type SignupHandler struct {
	userService service.UserService
	renderer    *handler.Renderer
}

// NewSignupHandler creates a new signup handler
func NewSignupHandler(userService service.UserService, renderer *handler.Renderer) *SignupHandler {
	return &SignupHandler{
		userService: userService,
		renderer:    renderer,
	}
}

// ServeHTTP handles GET /signup and POST /signup
func (h *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.showSignupForm(w, r, nil)
		return
	}

	if r.Method == http.MethodPost {
		h.handleSignup(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *SignupHandler) showSignupForm(w http.ResponseWriter, r *http.Request, formError *string) {
	data := BaseTemplateData(r)

	if formError != nil {
		data["Error"] = *formError
	}

	h.renderer.RenderHTTP(w, "signup", data)
}

func (h *SignupHandler) handleSignup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showSignupForm(w, r, &errMsg)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")

	// Validate required fields
	if email == "" || password == "" {
		errMsg := "Email and password are required"
		h.showSignupForm(w, r, &errMsg)
		return
	}

	// Register user
	user, err := h.userService.Register(ctx, email, password, firstName, lastName)
	if err != nil {
		var errMsg string
		if errors.Is(err, service.ErrUserExists) {
			errMsg = "An account with this email already exists"
		} else {
			errMsg = "Failed to create account. Please try again."
		}
		h.showSignupForm(w, r, &errMsg)
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
		Secure:   r.TLS != nil, // Only secure if using HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
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

// ServeHTTP handles GET /login and POST /login
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.showLoginForm(w, r, nil)
		return
	}

	if r.Method == http.MethodPost {
		h.handleLogin(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *LoginHandler) showLoginForm(w http.ResponseWriter, r *http.Request, formError *string) {
	data := BaseTemplateData(r)

	if formError != nil {
		data["Error"] = *formError
	}

	h.renderer.RenderHTTP(w, "login", data)
}

func (h *LoginHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showLoginForm(w, r, &errMsg)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate required fields
	if email == "" || password == "" {
		errMsg := "Email and password are required"
		h.showLoginForm(w, r, &errMsg)
		return
	}

	// Authenticate user
	user, err := h.userService.Authenticate(ctx, email, password)
	if err != nil {
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
		h.showLoginForm(w, r, &errMsg)
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

// ServeHTTP handles POST /logout
func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
