package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/tenant"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK RESOLVER
// =============================================================================

// mockResolver is a mock implementation of tenant.Resolver for testing.
type mockResolver struct {
	bySlugFunc         func(ctx context.Context, slug string) (*tenant.Tenant, error)
	byCustomDomainFunc func(ctx context.Context, domain string) (*tenant.Tenant, error)
	byIDFunc           func(ctx context.Context, id pgtype.UUID) (*tenant.Tenant, error)
}

func (m *mockResolver) BySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	if m.bySlugFunc != nil {
		return m.bySlugFunc(ctx, slug)
	}
	return nil, tenant.ErrTenantNotFound
}

func (m *mockResolver) ByCustomDomain(ctx context.Context, domain string) (*tenant.Tenant, error) {
	if m.byCustomDomainFunc != nil {
		return m.byCustomDomainFunc(ctx, domain)
	}
	return nil, tenant.ErrTenantNotFound
}

func (m *mockResolver) ByID(ctx context.Context, id pgtype.UUID) (*tenant.Tenant, error) {
	if m.byIDFunc != nil {
		return m.byIDFunc(ctx, id)
	}
	return nil, tenant.ErrTenantNotFound
}

// =============================================================================
// MOCK USER SERVICE
// =============================================================================

type mockUserService struct {
	getUserBySessionTokenFunc func(ctx context.Context, token string) (*domain.Customer, error)
}

func (m *mockUserService) GetUserBySessionToken(ctx context.Context, token string) (*domain.Customer, error) {
	if m.getUserBySessionTokenFunc != nil {
		return m.getUserBySessionTokenFunc(ctx, token)
	}
	return nil, &domain.Error{Code: domain.EUNAUTHORIZED, Message: "invalid session"}
}

// Stub implementations for other required methods

func (m *mockUserService) Register(ctx context.Context, email, password, firstName, lastName string) (*domain.Customer, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) Authenticate(ctx context.Context, email, password string) (*domain.Customer, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) CreateSession(ctx context.Context, userID string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockUserService) DeleteSession(ctx context.Context, token string) error {
	return errors.New("not implemented")
}

func (m *mockUserService) GetUserByID(ctx context.Context, userID string) (*domain.Customer, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) ListUsers(ctx context.Context, limit, offset int32) ([]domain.UserListItem, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) ListUsersByAccountType(ctx context.Context, accountType domain.UserAccountType) ([]domain.UserListItem, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) CountUsers(ctx context.Context) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockUserService) UpdateUserProfile(ctx context.Context, userID string, params domain.UpdateUserProfileParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) UpdateUserPassword(ctx context.Context, userID, newPassword string) error {
	return errors.New("not implemented")
}

func (m *mockUserService) UpdateUserStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	return errors.New("not implemented")
}

func (m *mockUserService) AdminUpdateCustomer(ctx context.Context, userID string, params domain.AdminUpdateCustomerParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) VerifyUserEmail(ctx context.Context, userID string) error {
	return errors.New("not implemented")
}

func (m *mockUserService) SubmitWholesaleApplication(ctx context.Context, userID string, params domain.WholesaleApplicationParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) UpdateWholesaleApplication(ctx context.Context, userID string, params domain.UpdateWholesaleApplicationParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) UpdateWholesaleCustomer(ctx context.Context, userID string, params domain.UpdateWholesaleCustomerParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) GetCustomersForBillingCycle(ctx context.Context, billingCycle domain.BillingCycle, day int32) ([]domain.Customer, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) ListAddresses(ctx context.Context, userID string) ([]domain.UserAddress, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) ListPaymentMethods(ctx context.Context, userID string) ([]domain.UserPaymentMethod, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) GetAccountSummary(ctx context.Context, userID string) (domain.AccountSummary, error) {
	return domain.AccountSummary{}, errors.New("not implemented")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func newTestUUID(id string) pgtype.UUID {
	var pgUUID pgtype.UUID
	// Simple test UUID - just use first 16 bytes of string padded with zeros
	copy(pgUUID.Bytes[:], []byte(id))
	pgUUID.Valid = true
	return pgUUID
}

func newActiveTenant(slug, name string) *tenant.Tenant {
	return &tenant.Tenant{
		ID:     newTestUUID(slug),
		Slug:   slug,
		Name:   name,
		Status: "active",
	}
}

// =============================================================================
// TESTS: ResolveTenant - Tenant Resolution
// =============================================================================

func Test_ResolveTenant_SubdomainResolution(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		resolveSlug     string
		tenant          *tenant.Tenant
		resolveErr      error
		expectTenant    bool
		expectStatus    int
		expectRedirect  string
		expectRetryAfter bool
	}{
		{
			name:         "subdomain resolves to active tenant",
			host:         "acme.hiri.coffee",
			resolveSlug:  "acme",
			tenant:       newActiveTenant("acme", "Acme Coffee"),
			expectTenant: true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "subdomain with port resolves correctly",
			host:         "acme.hiri.coffee:3000",
			resolveSlug:  "acme",
			tenant:       newActiveTenant("acme", "Acme Coffee"),
			expectTenant: true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "unknown subdomain returns 404",
			host:         "nonexistent.hiri.coffee",
			resolveSlug:  "nonexistent",
			resolveErr:   tenant.ErrTenantNotFound,
			expectTenant: false,
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "pending tenant returns 404",
			host:         "pending.hiri.coffee",
			resolveSlug:  "pending",
			tenant:       &tenant.Tenant{ID: newTestUUID("pending"), Slug: "pending", Name: "Pending", Status: "pending"},
			expectTenant: false,
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "suspended tenant returns 503 with retry-after",
			host:         "suspended.hiri.coffee",
			resolveSlug:  "suspended",
			tenant:       &tenant.Tenant{ID: newTestUUID("suspended"), Slug: "suspended", Name: "Suspended", Status: "suspended"},
			expectTenant: false,
			expectStatus: http.StatusServiceUnavailable,
			expectRetryAfter: true,
		},
		{
			name:         "cancelled tenant returns 404",
			host:         "cancelled.hiri.coffee",
			resolveSlug:  "cancelled",
			tenant:       &tenant.Tenant{ID: newTestUUID("cancelled"), Slug: "cancelled", Name: "Cancelled", Status: "cancelled"},
			expectTenant: false,
			expectStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockResolver{
				bySlugFunc: func(ctx context.Context, slug string) (*tenant.Tenant, error) {
					assert.Equal(t, tt.resolveSlug, slug, "resolver called with wrong slug")
					if tt.resolveErr != nil {
						return nil, tt.resolveErr
					}
					return tt.tenant, nil
				},
			}

			cfg := TenantConfig{
				BaseDomain: "hiri.coffee",
				AppDomain:  "app.hiri.coffee",
				Resolver:   resolver,
			}

			// Create test handler that checks context
			var handlerCalled bool
			var contextTenant *tenant.Tenant
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				contextTenant = tenant.FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			middleware := ResolveTenant(cfg)
			wrappedHandler := middleware(handler)

			// Make request
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			// Assertions
			assert.Equal(t, tt.expectStatus, rec.Code, "unexpected status code")

			if tt.expectTenant {
				require.True(t, handlerCalled, "handler should have been called")
				require.NotNil(t, contextTenant, "tenant should be in context")
				assert.Equal(t, tt.tenant.Slug, contextTenant.Slug)
				assert.Equal(t, tt.tenant.Status, contextTenant.Status)
			} else {
				assert.False(t, handlerCalled, "handler should not have been called")
			}

			if tt.expectRedirect != "" {
				location := rec.Header().Get("Location")
				assert.Equal(t, tt.expectRedirect, location, "redirect location mismatch")
			}

			if tt.expectRetryAfter {
				retryAfter := rec.Header().Get("Retry-After")
				assert.NotEmpty(t, retryAfter, "Retry-After header should be set for suspended tenant")
			}
		})
	}
}

func Test_ResolveTenant_CustomDomainResolution(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		resolveDomain string
		tenant       *tenant.Tenant
		resolveErr   error
		expectTenant bool
		expectStatus int
	}{
		{
			name:          "custom domain resolves to active tenant",
			host:          "shop.acme.com",
			resolveDomain: "shop.acme.com",
			tenant:        newActiveTenant("acme", "Acme Coffee"),
			expectTenant:  true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "custom domain with port",
			host:          "shop.acme.com:8080",
			resolveDomain: "shop.acme.com",
			tenant:        newActiveTenant("acme", "Acme Coffee"),
			expectTenant:  true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "unknown custom domain returns 404",
			host:          "unknown.com",
			resolveDomain: "unknown.com",
			resolveErr:    tenant.ErrTenantNotFound,
			expectTenant:  false,
			expectStatus:  http.StatusNotFound,
		},
		{
			name:          "inactive custom domain returns 404",
			host:          "inactive.acme.com",
			resolveDomain: "inactive.acme.com",
			resolveErr:    tenant.ErrCustomDomainNotActive,
			expectTenant:  false,
			expectStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockResolver{
				byCustomDomainFunc: func(ctx context.Context, domain string) (*tenant.Tenant, error) {
					assert.Equal(t, tt.resolveDomain, domain, "resolver called with wrong domain")
					if tt.resolveErr != nil {
						return nil, tt.resolveErr
					}
					return tt.tenant, nil
				},
			}

			cfg := TenantConfig{
				BaseDomain: "hiri.coffee",
				AppDomain:  "app.hiri.coffee",
				Resolver:   resolver,
			}

			var handlerCalled bool
			var contextTenant *tenant.Tenant
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				contextTenant = tenant.FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			middleware := ResolveTenant(cfg)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectStatus, rec.Code)

			if tt.expectTenant {
				require.True(t, handlerCalled)
				require.NotNil(t, contextTenant)
				assert.Equal(t, tt.tenant.Slug, contextTenant.Slug)
			} else {
				assert.False(t, handlerCalled)
			}
		})
	}
}

func Test_ResolveTenant_SpecialDomains(t *testing.T) {
	tests := []struct {
		name                string
		host                string
		expectResolverCall  bool
		expectHandlerCall   bool
		expectStatus        int
		expectRedirectTo    string
	}{
		{
			name:               "app.hiri.coffee bypasses tenant resolution",
			host:               "app.hiri.coffee",
			expectResolverCall: false,
			expectHandlerCall:  true,
			expectStatus:       http.StatusOK,
		},
		{
			name:               "app subdomain with port bypasses resolution",
			host:               "app.hiri.coffee:3000",
			expectResolverCall: false,
			expectHandlerCall:  true,
			expectStatus:       http.StatusOK,
		},
		{
			name:               "base domain apex bypasses resolution",
			host:               "hiri.coffee",
			expectResolverCall: false,
			expectHandlerCall:  true,
			expectStatus:       http.StatusOK,
		},
		{
			name:               "base domain with port bypasses resolution",
			host:               "hiri.coffee:3000",
			expectResolverCall: false,
			expectHandlerCall:  true,
			expectStatus:       http.StatusOK,
		},
		{
			name:               "www subdomain redirects to apex",
			host:               "www.hiri.coffee",
			expectResolverCall: false,
			expectHandlerCall:  false,
			expectStatus:       http.StatusMovedPermanently,
			expectRedirectTo:   "https://hiri.coffee/",
		},
		{
			name:               "www redirect preserves path and query",
			host:               "www.hiri.coffee",
			expectResolverCall: false,
			expectHandlerCall:  false,
			expectStatus:       http.StatusMovedPermanently,
			expectRedirectTo:   "https://hiri.coffee/products?category=single-origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resolverCalled bool
			resolver := &mockResolver{
				bySlugFunc: func(ctx context.Context, slug string) (*tenant.Tenant, error) {
					resolverCalled = true
					return nil, tenant.ErrTenantNotFound
				},
				byCustomDomainFunc: func(ctx context.Context, domain string) (*tenant.Tenant, error) {
					resolverCalled = true
					return nil, tenant.ErrTenantNotFound
				},
			}

			cfg := TenantConfig{
				BaseDomain: "hiri.coffee",
				AppDomain:  "app.hiri.coffee",
				Resolver:   resolver,
			}

			var handlerCalled bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			middleware := ResolveTenant(cfg)
			wrappedHandler := middleware(handler)

			path := "/"
			if tt.expectRedirectTo != "" && tt.expectRedirectTo != "https://hiri.coffee/" {
				// Extract path from redirect URL for www test
				path = "/products?category=single-origin"
			}
			req := httptest.NewRequest("GET", path, nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectResolverCall, resolverCalled, "resolver call expectation mismatch")
			assert.Equal(t, tt.expectHandlerCall, handlerCalled, "handler call expectation mismatch")
			assert.Equal(t, tt.expectStatus, rec.Code, "status code mismatch")

			if tt.expectRedirectTo != "" {
				location := rec.Header().Get("Location")
				assert.Equal(t, tt.expectRedirectTo, location, "redirect location mismatch")
			}
		})
	}
}

func Test_ResolveTenant_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		resolveErr    error
		expectStatus  int
		expectBody    string
	}{
		{
			name:         "tenant not found returns 404",
			host:         "missing.hiri.coffee",
			resolveErr:   tenant.ErrTenantNotFound,
			expectStatus: http.StatusNotFound,
			expectBody:   "Not Found",
		},
		{
			name:         "custom domain not active returns 404",
			host:         "unverified.acme.com",
			resolveErr:   tenant.ErrCustomDomainNotActive,
			expectStatus: http.StatusNotFound,
			expectBody:   "Not Found",
		},
		{
			name:         "database error returns 500",
			host:         "error.hiri.coffee",
			resolveErr:   errors.New("database connection failed"),
			expectStatus: http.StatusInternalServerError,
			expectBody:   "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockResolver{
				bySlugFunc: func(ctx context.Context, slug string) (*tenant.Tenant, error) {
					return nil, tt.resolveErr
				},
				byCustomDomainFunc: func(ctx context.Context, domain string) (*tenant.Tenant, error) {
					return nil, tt.resolveErr
				},
			}

			cfg := TenantConfig{
				BaseDomain: "hiri.coffee",
				AppDomain:  "app.hiri.coffee",
				Resolver:   resolver,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not be called on error")
			})

			middleware := ResolveTenant(cfg)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.expectBody)
		})
	}
}

// =============================================================================
// TESTS: RequireTenant Middleware
// =============================================================================

func Test_RequireTenant(t *testing.T) {
	tests := []struct {
		name             string
		tenantInContext  *tenant.Tenant
		expectHandlerRun bool
		expectStatus     int
	}{
		{
			name:             "tenant present in context - continues",
			tenantInContext:  newActiveTenant("acme", "Acme Coffee"),
			expectHandlerRun: true,
			expectStatus:     http.StatusOK,
		},
		{
			name:             "no tenant in context - returns 404",
			tenantInContext:  nil,
			expectHandlerRun: false,
			expectStatus:     http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := RequireTenant(handler)

			ctx := context.Background()
			if tt.tenantInContext != nil {
				ctx = tenant.NewContext(ctx, tt.tenantInContext)
			}

			req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectHandlerRun, handlerCalled)
			assert.Equal(t, tt.expectStatus, rec.Code)
		})
	}
}

// =============================================================================
// TESTS: User Authentication (auth.go)
// =============================================================================

func Test_WithUser_SessionExtraction(t *testing.T) {
	tests := []struct {
		name              string
		sessionToken      string
		mockUser          *domain.Customer
		mockErr           error
		expectUserInCtx   bool
		expectHandlerCall bool
	}{
		{
			name:         "valid session extracts user",
			sessionToken: "valid_session_token",
			mockUser: &domain.Customer{
				ID:          newTestUUID("user1"),
				Email:       "user@example.com",
				AccountType: domain.UserAccountTypeRetail,
			},
			expectUserInCtx:   true,
			expectHandlerCall: true,
		},
		{
			name:              "missing session cookie continues without user",
			sessionToken:      "",
			expectUserInCtx:   false,
			expectHandlerCall: true,
		},
		{
			name:              "invalid session continues without user",
			sessionToken:      "invalid_token",
			mockErr:           &domain.Error{Code: domain.EUNAUTHORIZED, Message: "invalid session"},
			expectUserInCtx:   false,
			expectHandlerCall: true,
		},
		{
			name:              "expired session continues without user",
			sessionToken:      "expired_token",
			mockErr:           &domain.Error{Code: domain.EUNAUTHORIZED, Message: "session expired"},
			expectUserInCtx:   false,
			expectHandlerCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userService := &mockUserService{
				getUserBySessionTokenFunc: func(ctx context.Context, token string) (*domain.Customer, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return tt.mockUser, nil
				},
			}

			var handlerCalled bool
			var userFromContext *domain.Customer
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				userFromContext = GetUserFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			middleware := WithUser(userService)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/", nil)
			if tt.sessionToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  sessionCookieName,
					Value: tt.sessionToken,
				})
			}
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectHandlerCall, handlerCalled)

			if tt.expectUserInCtx {
				require.NotNil(t, userFromContext, "expected user in context")
				assert.Equal(t, tt.mockUser.Email, userFromContext.Email)
			} else {
				assert.Nil(t, userFromContext, "expected no user in context")
			}
		})
	}
}

func Test_RequireAuth_RedirectsWhenNotAuthenticated(t *testing.T) {
	tests := []struct {
		name               string
		user               *domain.Customer
		requestPath        string
		requestQuery       string
		expectRedirect     bool
		expectRedirectTo   string
		expectHandlerCall  bool
	}{
		{
			name: "authenticated user continues",
			user: &domain.Customer{
				ID:          newTestUUID("user1"),
				Email:       "user@example.com",
				AccountType: domain.UserAccountTypeRetail,
			},
			requestPath:       "/account",
			expectRedirect:    false,
			expectHandlerCall: true,
		},
		{
			name:              "no user redirects to login with return URL",
			user:              nil,
			requestPath:       "/account/orders",
			expectRedirect:    true,
			expectRedirectTo:  "/login?return_to=%2Faccount%2Forders",
			expectHandlerCall: false,
		},
		{
			name:              "redirect preserves query parameters",
			user:              nil,
			requestPath:       "/checkout",
			requestQuery:      "step=payment",
			expectRedirect:    true,
			expectRedirectTo:  "/login?return_to=%2Fcheckout%3Fstep%3Dpayment",
			expectHandlerCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := RequireAuth(handler)

			ctx := context.Background()
			if tt.user != nil {
				ctx = context.WithValue(ctx, UserContextKey, tt.user)
			}

			url := tt.requestPath
			if tt.requestQuery != "" {
				url += "?" + tt.requestQuery
			}
			req := httptest.NewRequest("GET", url, nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectHandlerCall, handlerCalled)

			if tt.expectRedirect {
				assert.Equal(t, http.StatusSeeOther, rec.Code)
				location := rec.Header().Get("Location")
				assert.Equal(t, tt.expectRedirectTo, location)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

func Test_RequireAdmin_RedirectsNonAdmins(t *testing.T) {
	tests := []struct {
		name              string
		user              *domain.Customer
		expectRedirectTo  string
		expectHandlerCall bool
	}{
		{
			name: "admin user continues",
			user: &domain.Customer{
				ID:          newTestUUID("admin1"),
				Email:       "admin@example.com",
				AccountType: domain.UserAccountTypeAdmin,
			},
			expectHandlerCall: true,
		},
		{
			name:             "no user redirects to admin login",
			user:             nil,
			expectRedirectTo: "/admin/login",
			expectHandlerCall: false,
		},
		{
			name: "retail user redirects to storefront",
			user: &domain.Customer{
				ID:          newTestUUID("user1"),
				Email:       "user@example.com",
				AccountType: domain.UserAccountTypeRetail,
			},
			expectRedirectTo: "/",
			expectHandlerCall: false,
		},
		{
			name: "wholesale user redirects to storefront",
			user: &domain.Customer{
				ID:          newTestUUID("wholesale1"),
				Email:       "wholesale@example.com",
				AccountType: domain.UserAccountTypeWholesale,
			},
			expectRedirectTo: "/",
			expectHandlerCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := RequireAdmin(handler)

			ctx := context.Background()
			if tt.user != nil {
				ctx = context.WithValue(ctx, UserContextKey, tt.user)
			}

			req := httptest.NewRequest("GET", "/admin/dashboard", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectHandlerCall, handlerCalled)

			if tt.expectRedirectTo != "" {
				assert.Equal(t, http.StatusSeeOther, rec.Code)
				location := rec.Header().Get("Location")
				assert.Equal(t, tt.expectRedirectTo, location)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// =============================================================================
// TESTS: Helper Functions
// =============================================================================

func Test_extractSubdomainForTenant(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		baseDomain string
		expected   string
	}{
		{
			name:       "extracts single-level subdomain",
			host:       "acme.hiri.coffee",
			baseDomain: "hiri.coffee",
			expected:   "acme",
		},
		{
			name:       "apex domain returns empty string",
			host:       "hiri.coffee",
			baseDomain: "hiri.coffee",
			expected:   "",
		},
		{
			name:       "custom domain returns empty string",
			host:       "shop.example.com",
			baseDomain: "hiri.coffee",
			expected:   "",
		},
		{
			name:       "nested subdomain returns empty string",
			host:       "sub.acme.hiri.coffee",
			baseDomain: "hiri.coffee",
			expected:   "",
		},
		{
			name:       "different TLD returns empty string",
			host:       "acme.hiri.com",
			baseDomain: "hiri.coffee",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubdomainForTenant(tt.host, tt.baseDomain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_stripPort(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "removes port from host",
			host:     "acme.hiri.coffee:3000",
			expected: "acme.hiri.coffee",
		},
		{
			name:     "no port returns unchanged",
			host:     "acme.hiri.coffee",
			expected: "acme.hiri.coffee",
		},
		{
			name:     "localhost with port",
			host:     "localhost:8080",
			expected: "localhost",
		},
		{
			name:     "empty string returns empty",
			host:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripPort(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// INTEGRATION TEST: Full Middleware Chain
// =============================================================================

func Test_FullMiddlewareChain_TenantAndAuth(t *testing.T) {
	// This test verifies that ResolveTenant → WithUser → RequireAuth works correctly
	activeTenant := newActiveTenant("acme", "Acme Coffee")
	authenticatedUser := &domain.Customer{
		ID:          newTestUUID("user1"),
		Email:       "user@acme.com",
		AccountType: domain.UserAccountTypeRetail,
	}

	resolver := &mockResolver{
		bySlugFunc: func(ctx context.Context, slug string) (*tenant.Tenant, error) {
			if slug == "acme" {
				return activeTenant, nil
			}
			return nil, tenant.ErrTenantNotFound
		},
	}

	userService := &mockUserService{
		getUserBySessionTokenFunc: func(ctx context.Context, token string) (*domain.Customer, error) {
			if token == "valid_token" {
				return authenticatedUser, nil
			}
			return nil, &domain.Error{Code: domain.EUNAUTHORIZED}
		},
	}

	// Build middleware chain
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify both tenant and user are in context
		tenantCtx := tenant.FromContext(r.Context())
		require.NotNil(t, tenantCtx, "tenant should be in context")
		assert.Equal(t, "acme", tenantCtx.Slug)

		u := GetUserFromContext(r.Context())
		require.NotNil(t, u, "user should be in context")
		assert.Equal(t, authenticatedUser.Email, u.Email)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Chain: ResolveTenant → WithUser → RequireAuth → handler
	wrappedHandler := ResolveTenant(TenantConfig{
		BaseDomain: "hiri.coffee",
		AppDomain:  "app.hiri.coffee",
		Resolver:   resolver,
	})(WithUser(userService)(RequireAuth(handler)))

	// Test with valid tenant and valid session
	req := httptest.NewRequest("GET", "/account", nil)
	req.Host = "acme.hiri.coffee"
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "valid_token",
	})
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "success")
}

func Test_FullMiddlewareChain_RequireTenant(t *testing.T) {
	// Test that RequireTenant correctly blocks requests without tenant context
	activeTenant := newActiveTenant("acme", "Acme Coffee")

	resolver := &mockResolver{
		bySlugFunc: func(ctx context.Context, slug string) (*tenant.Tenant, error) {
			if slug == "acme" {
				return activeTenant, nil
			}
			return nil, tenant.ErrTenantNotFound
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantCtx := tenant.FromContext(r.Context())
		require.NotNil(t, tenantCtx, "tenant must be in context after RequireTenant")
		w.WriteHeader(http.StatusOK)
	})

	// Chain: ResolveTenant → RequireTenant → handler
	wrappedHandler := ResolveTenant(TenantConfig{
		BaseDomain: "hiri.coffee",
		AppDomain:  "app.hiri.coffee",
		Resolver:   resolver,
	})(RequireTenant(handler))

	// Test 1: Valid tenant - should work
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "acme.hiri.coffee"
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Test 2: No tenant (app domain) - should return 404
	req = httptest.NewRequest("GET", "/", nil)
	req.Host = "app.hiri.coffee"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
