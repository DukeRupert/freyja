package storefront

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dukerupert/freyja/internal/service"
)

// mockCartService implements service.CartService for testing
type mockCartService struct {
	getOrCreateCartFunc    func(ctx context.Context, sessionID string) (*service.Cart, string, error)
	getCartFunc            func(ctx context.Context, sessionID string) (*service.Cart, error)
	addItemFunc            func(ctx context.Context, cartID string, skuID string, quantity int) (*service.CartSummary, error)
	updateItemQuantityFunc func(ctx context.Context, cartID string, skuID string, quantity int) (*service.CartSummary, error)
	removeItemFunc         func(ctx context.Context, cartID string, skuID string) (*service.CartSummary, error)
	getCartSummaryFunc     func(ctx context.Context, cartID string) (*service.CartSummary, error)
}

func (m *mockCartService) GetOrCreateCart(ctx context.Context, sessionID string) (*service.Cart, string, error) {
	if m.getOrCreateCartFunc != nil {
		return m.getOrCreateCartFunc(ctx, sessionID)
	}
	return nil, "", nil
}

func (m *mockCartService) GetCart(ctx context.Context, sessionID string) (*service.Cart, error) {
	if m.getCartFunc != nil {
		return m.getCartFunc(ctx, sessionID)
	}
	return nil, nil
}

func (m *mockCartService) AddItem(ctx context.Context, cartID string, skuID string, quantity int) (*service.CartSummary, error) {
	if m.addItemFunc != nil {
		return m.addItemFunc(ctx, cartID, skuID, quantity)
	}
	return nil, nil
}

func (m *mockCartService) UpdateItemQuantity(ctx context.Context, cartID string, skuID string, quantity int) (*service.CartSummary, error) {
	if m.updateItemQuantityFunc != nil {
		return m.updateItemQuantityFunc(ctx, cartID, skuID, quantity)
	}
	return nil, nil
}

func (m *mockCartService) RemoveItem(ctx context.Context, cartID string, skuID string) (*service.CartSummary, error) {
	if m.removeItemFunc != nil {
		return m.removeItemFunc(ctx, cartID, skuID)
	}
	return nil, nil
}

func (m *mockCartService) GetCartSummary(ctx context.Context, cartID string) (*service.CartSummary, error) {
	if m.getCartSummaryFunc != nil {
		return m.getCartSummaryFunc(ctx, cartID)
	}
	return nil, nil
}

// Test CartViewHandler.ServeHTTP
func TestCartViewHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		sessionCookie  string
		mockCart       *service.Cart
		mockSummary    *service.CartSummary
		mockGetError   error
		mockSummaryErr error
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "no session cookie shows empty cart",
			sessionCookie:  "",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Your cart is empty") {
					t.Error("expected empty cart message")
				}
				if !strings.Contains(body, "Continue Shopping") {
					t.Error("expected continue shopping link")
				}
			},
		},
		{
			name:           "session exists but cart not found shows empty",
			sessionCookie:  "valid-session-id",
			mockCart:       nil,
			mockGetError:   service.ErrCartNotFound,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Your cart is empty") {
					t.Error("expected empty cart message")
				}
			},
		},
		{
			name:          "cart exists with items",
			sessionCookie: "valid-session-id",
			mockCart: &service.Cart{
				ID:        mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
				TenantID:  mustParseUUID("223e4567-e89b-12d3-a456-426614174000"),
				SessionID: mustParseUUID("323e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				Cart: service.Cart{
					ID: mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
				},
				Items: []service.CartItem{
					{
						ID:             mustParseUUID("423e4567-e89b-12d3-a456-426614174000"),
						ProductName:    "Ethiopian Yirgacheffe",
						SKU:            "ETH-YIR-12OZ-WHOLE",
						WeightValue:    "12oz",
						Grind:          "Whole Bean",
						Quantity:       2,
						UnitPriceCents: 1599,
						LineSubtotal:   3198,
					},
					{
						ID:             mustParseUUID("423e4567-e89b-12d3-a456-426614174001"),
						ProductName:    "Colombian Supremo",
						SKU:            "COL-SUP-16OZ-GROUND",
						WeightValue:    "16oz",
						Grind:          "Ground",
						Quantity:       1,
						UnitPriceCents: 1899,
						LineSubtotal:   1899,
					},
				},
				Subtotal:  5097,
				ItemCount: 3,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "3 items") {
					t.Error("expected item count of 3")
				}
				if !strings.Contains(body, "Ethiopian Yirgacheffe") {
					t.Error("expected product name")
				}
				if !strings.Contains(body, "12oz") {
					t.Error("expected weight value")
				}
				if !strings.Contains(body, "Whole Bean") {
					t.Error("expected grind type")
				}
				if !strings.Contains(body, "x 2") {
					t.Error("expected quantity display")
				}
				if !strings.Contains(body, "$31.98") {
					t.Error("expected line item subtotal for first item")
				}
				if !strings.Contains(body, "$50.97") {
					t.Error("expected cart subtotal")
				}
				if !strings.Contains(body, "Colombian Supremo") {
					t.Error("expected second product")
				}
			},
		},
		{
			name:           "service error on GetCart returns 500",
			sessionCookie:  "valid-session-id",
			mockCart:       nil,
			mockGetError:   errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to load cart") {
					t.Error("expected error message")
				}
			},
		},
		{
			name:          "service error on GetCartSummary returns 500",
			sessionCookie: "valid-session-id",
			mockCart: &service.Cart{
				ID: mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary:    nil,
			mockSummaryErr: errors.New("failed to fetch items"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to load cart details") {
					t.Error("expected error message")
				}
			},
		},
		{
			name:          "cart with zero items shows subtotal correctly",
			sessionCookie: "valid-session-id",
			mockCart: &service.Cart{
				ID: mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				Items:     []service.CartItem{},
				Subtotal:  0,
				ItemCount: 0,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "0 items") {
					t.Error("expected 0 items")
				}
				if !strings.Contains(body, "$0.00") {
					t.Error("expected $0.00 subtotal")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCartService{
				getCartFunc: func(ctx context.Context, sessionID string) (*service.Cart, error) {
					if tt.sessionCookie != "" && sessionID != tt.sessionCookie {
						t.Errorf("expected sessionID %q, got %q", tt.sessionCookie, sessionID)
					}
					return tt.mockCart, tt.mockGetError
				},
				getCartSummaryFunc: func(ctx context.Context, cartID string) (*service.CartSummary, error) {
					return tt.mockSummary, tt.mockSummaryErr
				},
			}

			handler := NewCartViewHandler(mock, nil, false)

			req := httptest.NewRequest(http.MethodGet, "/cart", nil)
			if tt.sessionCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "freyja_session",
					Value: tt.sessionCookie,
				})
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}

// Test AddToCartHandler.ServeHTTP
func TestAddToCartHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name               string
		formData           url.Values
		sessionCookie      string
		mockCart           *service.Cart
		mockNewSessionID   string
		mockSummary        *service.CartSummary
		mockGetOrCreateErr error
		mockAddItemErr     error
		expectedStatus     int
		expectCookieSet    bool
		checkBody          func(t *testing.T, body string)
		checkCookie        func(t *testing.T, cookies []*http.Cookie)
	}{
		{
			name: "successfully add item to new cart",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"2"},
			},
			sessionCookie:    "",
			mockNewSessionID: "new-session-id-12345",
			mockCart: &service.Cart{
				ID: mustParseUUID("523e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 2,
				Subtotal:  3198,
			},
			expectedStatus:  http.StatusOK,
			expectCookieSet: true,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Item added!") {
					t.Error("expected success message")
				}
				if !strings.Contains(body, "2 items") {
					t.Error("expected item count")
				}
				if !strings.Contains(body, "$31.98") {
					t.Error("expected subtotal")
				}
			},
			checkCookie: func(t *testing.T, cookies []*http.Cookie) {
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "freyja_session" {
						found = true
						if cookie.Value != "new-session-id-12345" {
							t.Errorf("expected cookie value %q, got %q", "new-session-id-12345", cookie.Value)
						}
					}
				}
				if !found {
					t.Error("expected session cookie to be set")
				}
			},
		},
		{
			name: "add item to existing cart",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"1"},
			},
			sessionCookie:    "existing-session",
			mockNewSessionID: "existing-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("523e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 3,
				Subtotal:  5097,
			},
			expectedStatus:  http.StatusOK,
			expectCookieSet: false,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "3 items") {
					t.Error("expected updated item count")
				}
				if !strings.Contains(body, "$50.97") {
					t.Error("expected updated subtotal")
				}
			},
		},
		{
			name: "invalid form data returns 400",
			formData: url.Values{
				"invalid": []string{"data"},
			},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
		{
			name: "missing sku_id returns 400",
			formData: url.Values{
				"quantity": []string{"1"},
			},
			mockNewSessionID: "new-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("523e4567-e89b-12d3-a456-426614174000"),
			},
			mockAddItemErr: service.ErrSKUNotFound,
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Product not found") {
					t.Error("expected product not found error")
				}
			},
		},
		{
			name: "invalid quantity zero returns 400",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"0"},
			},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
		{
			name: "invalid quantity negative returns 400",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"-5"},
			},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
		{
			name: "invalid quantity non-numeric returns 400",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"abc"},
			},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
		{
			name: "SKU not found returns 404",
			formData: url.Values{
				"sku_id":   []string{"nonexistent-sku"},
				"quantity": []string{"1"},
			},
			mockNewSessionID: "new-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("523e4567-e89b-12d3-a456-426614174000"),
			},
			mockAddItemErr: service.ErrSKUNotFound,
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Product not found") {
					t.Error("expected product not found error")
				}
			},
		},
		{
			name: "service error on GetOrCreateCart returns 500",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"1"},
			},
			mockGetOrCreateErr: errors.New("database error"),
			expectedStatus:     http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Cart error") {
					t.Error("expected cart error message")
				}
			},
		},
		{
			name: "service error on AddItem returns 500",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"1"},
			},
			mockNewSessionID: "session",
			mockCart: &service.Cart{
				ID: mustParseUUID("523e4567-e89b-12d3-a456-426614174000"),
			},
			mockAddItemErr: errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to add item") {
					t.Error("expected failed to add item error")
				}
			},
		},
		{
			name: "service returns ErrInvalidQuantity returns 400",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"1"},
			},
			mockNewSessionID: "session",
			mockCart: &service.Cart{
				ID: mustParseUUID("523e4567-e89b-12d3-a456-426614174000"),
			},
			mockAddItemErr: service.ErrInvalidQuantity,
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCartService{
				getOrCreateCartFunc: func(ctx context.Context, sessionID string) (*service.Cart, string, error) {
					return tt.mockCart, tt.mockNewSessionID, tt.mockGetOrCreateErr
				},
				addItemFunc: func(ctx context.Context, cartID string, skuID string, quantity int) (*service.CartSummary, error) {
					return tt.mockSummary, tt.mockAddItemErr
				},
			}

			handler := NewAddToCartHandler(mock, nil, false)

			req := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tt.sessionCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "freyja_session",
					Value: tt.sessionCookie,
				})
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}

			if tt.checkCookie != nil {
				tt.checkCookie(t, w.Result().Cookies())
			}
		})
	}
}

// Test UpdateCartItemHandler.ServeHTTP
func TestUpdateCartItemHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		formData       url.Values
		sessionCookie  string
		mockCart       *service.Cart
		mockSummary    *service.CartSummary
		mockGetErr     error
		mockUpdateErr  error
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "successfully update item quantity",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"5"},
			},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("623e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 5,
				Subtotal:  7995,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Cart updated!") {
					t.Error("expected update message")
				}
				if !strings.Contains(body, "5 items") {
					t.Error("expected updated item count")
				}
				if !strings.Contains(body, "$79.95") {
					t.Error("expected updated subtotal")
				}
			},
		},
		{
			name: "update quantity to zero removes item",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"0"},
			},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("623e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 2,
				Subtotal:  3198,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Cart updated!") {
					t.Error("expected update message")
				}
			},
		},
		{
			name: "no session cookie returns 404",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"2"},
			},
			sessionCookie:  "",
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "No cart found") {
					t.Error("expected no cart found error")
				}
			},
		},
		{
			name: "invalid quantity negative returns 400",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"-1"},
			},
			sessionCookie:  "valid-session",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
		{
			name: "invalid quantity non-numeric returns 400",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"xyz"},
			},
			sessionCookie:  "valid-session",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid quantity") {
					t.Error("expected invalid quantity error")
				}
			},
		},
		{
			name: "cart not found returns 404",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"2"},
			},
			sessionCookie:  "invalid-session",
			mockCart:       nil,
			mockGetErr:     service.ErrCartNotFound,
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Cart not found") {
					t.Error("expected cart not found error")
				}
			},
		},
		{
			name: "service error on GetCart returns 404",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"2"},
			},
			sessionCookie:  "valid-session",
			mockGetErr:     errors.New("database error"),
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Cart not found") {
					t.Error("expected cart not found error")
				}
			},
		},
		{
			name: "service error on UpdateItemQuantity returns 500",
			formData: url.Values{
				"sku_id":   []string{"123e4567-e89b-12d3-a456-426614174000"},
				"quantity": []string{"2"},
			},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("623e4567-e89b-12d3-a456-426614174000"),
			},
			mockUpdateErr:  errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to update item") {
					t.Error("expected failed to update item error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCartService{
				getCartFunc: func(ctx context.Context, sessionID string) (*service.Cart, error) {
					return tt.mockCart, tt.mockGetErr
				},
				updateItemQuantityFunc: func(ctx context.Context, cartID string, skuID string, quantity int) (*service.CartSummary, error) {
					return tt.mockSummary, tt.mockUpdateErr
				},
			}

			handler := NewUpdateCartItemHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/cart/update", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tt.sessionCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "freyja_session",
					Value: tt.sessionCookie,
				})
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}

// Test RemoveCartItemHandler.ServeHTTP
func TestRemoveCartItemHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		formData       url.Values
		sessionCookie  string
		mockCart       *service.Cart
		mockSummary    *service.CartSummary
		mockGetErr     error
		mockRemoveErr  error
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "successfully remove item",
			formData: url.Values{
				"sku_id": []string{"123e4567-e89b-12d3-a456-426614174000"},
			},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("723e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 1,
				Subtotal:  1599,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Item removed!") {
					t.Error("expected item removed message")
				}
				if !strings.Contains(body, "1 items") {
					t.Error("expected updated item count")
				}
				if !strings.Contains(body, "$15.99") {
					t.Error("expected updated subtotal")
				}
			},
		},
		{
			name: "remove last item shows zero",
			formData: url.Values{
				"sku_id": []string{"123e4567-e89b-12d3-a456-426614174000"},
			},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("723e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 0,
				Subtotal:  0,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "0 items") {
					t.Error("expected 0 items")
				}
				if !strings.Contains(body, "$0.00") {
					t.Error("expected $0.00 subtotal")
				}
			},
		},
		{
			name: "no session cookie returns 404",
			formData: url.Values{
				"sku_id": []string{"123e4567-e89b-12d3-a456-426614174000"},
			},
			sessionCookie:  "",
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "No cart found") {
					t.Error("expected no cart found error")
				}
			},
		},
		{
			name:          "missing sku_id still processes with empty string",
			formData:      url.Values{},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("723e4567-e89b-12d3-a456-426614174000"),
			},
			mockSummary: &service.CartSummary{
				ItemCount: 1,
				Subtotal:  1599,
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Item removed!") {
					t.Error("expected item removed message even with empty sku_id")
				}
			},
		},
		{
			name: "cart not found returns 404",
			formData: url.Values{
				"sku_id": []string{"123e4567-e89b-12d3-a456-426614174000"},
			},
			sessionCookie:  "invalid-session",
			mockGetErr:     service.ErrCartNotFound,
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Cart not found") {
					t.Error("expected cart not found error")
				}
			},
		},
		{
			name: "service error on RemoveItem returns 500",
			formData: url.Values{
				"sku_id": []string{"123e4567-e89b-12d3-a456-426614174000"},
			},
			sessionCookie: "valid-session",
			mockCart: &service.Cart{
				ID: mustParseUUID("723e4567-e89b-12d3-a456-426614174000"),
			},
			mockRemoveErr:  errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to remove item") {
					t.Error("expected failed to remove item error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCartService{
				getCartFunc: func(ctx context.Context, sessionID string) (*service.Cart, error) {
					return tt.mockCart, tt.mockGetErr
				},
				removeItemFunc: func(ctx context.Context, cartID string, skuID string) (*service.CartSummary, error) {
					return tt.mockSummary, tt.mockRemoveErr
				},
			}

			handler := NewRemoveCartItemHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/cart/remove", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tt.sessionCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "freyja_session",
					Value: tt.sessionCookie,
				})
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}
