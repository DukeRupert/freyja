package storefront

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockProductService implements service.ProductService for testing
type mockProductService struct {
	listProductsFunc     func(ctx context.Context) ([]repository.ListActiveProductsRow, error)
	getProductDetailFunc func(ctx context.Context, slug string) (*service.ProductDetail, error)
	getProductPriceFunc  func(ctx context.Context, skuID string) (*service.ProductPrice, error)
}

func (m *mockProductService) ListProducts(ctx context.Context) ([]repository.ListActiveProductsRow, error) {
	if m.listProductsFunc != nil {
		return m.listProductsFunc(ctx)
	}
	return nil, nil
}

func (m *mockProductService) GetProductDetail(ctx context.Context, slug string) (*service.ProductDetail, error) {
	if m.getProductDetailFunc != nil {
		return m.getProductDetailFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockProductService) GetProductPrice(ctx context.Context, skuID string) (*service.ProductPrice, error) {
	if m.getProductPriceFunc != nil {
		return m.getProductPriceFunc(ctx, skuID)
	}
	return nil, nil
}

// Test ProductListHandler.ServeHTTP
func TestProductListHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		mockProducts   []repository.ListActiveProductsRow
		mockError      error
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "success with multiple products",
			mockProducts: []repository.ListActiveProductsRow{
				{
					ID:   mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
					Name: "Ethiopian Yirgacheffe",
					Slug: "ethiopian-yirgacheffe",
					Origin: pgtype.Text{
						String: "Ethiopia",
						Valid:  true,
					},
					RoastLevel: pgtype.Text{
						String: "Light",
						Valid:  true,
					},
				},
				{
					ID:   mustParseUUID("123e4567-e89b-12d3-a456-426614174001"),
					Name: "Colombian Supremo",
					Slug: "colombian-supremo",
					Origin: pgtype.Text{
						String: "Colombia",
						Valid:  true,
					},
					RoastLevel: pgtype.Text{
						String: "Medium",
						Valid:  true,
					},
				},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Ethiopian Yirgacheffe") {
					t.Error("expected body to contain 'Ethiopian Yirgacheffe'")
				}
				if !strings.Contains(body, "Colombian Supremo") {
					t.Error("expected body to contain 'Colombian Supremo'")
				}
				if !strings.Contains(body, "Found 2 products") {
					t.Error("expected body to show product count of 2")
				}
				if !strings.Contains(body, "Ethiopia") {
					t.Error("expected body to contain origin 'Ethiopia'")
				}
				if !strings.Contains(body, `href="/products/ethiopian-yirgacheffe"`) {
					t.Error("expected body to contain link to product detail page")
				}
			},
		},
		{
			name:           "success with empty product list",
			mockProducts:   []repository.ListActiveProductsRow{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Found 0 products") {
					t.Error("expected body to show product count of 0")
				}
				if !strings.Contains(body, "Products") {
					t.Error("expected body to contain 'Products' heading")
				}
			},
		},
		{
			name:           "service error returns 500",
			mockProducts:   nil,
			mockError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to load products") {
					t.Error("expected body to contain error message")
				}
			},
		},
		{
			name: "handles products with null origin gracefully",
			mockProducts: []repository.ListActiveProductsRow{
				{
					ID:   mustParseUUID("123e4567-e89b-12d3-a456-426614174002"),
					Name: "Mystery Blend",
					Slug: "mystery-blend",
					Origin: pgtype.Text{
						Valid: false,
					},
				},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Mystery Blend") {
					t.Error("expected body to contain 'Mystery Blend'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				listProductsFunc: func(ctx context.Context) ([]repository.ListActiveProductsRow, error) {
					return tt.mockProducts, tt.mockError
				},
			}

			handler := NewProductListHandler(mock, nil)

			req := httptest.NewRequest(http.MethodGet, "/products", nil)
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

// Test ProductDetailHandler.ServeHTTP
func TestProductDetailHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		slug           string
		mockDetail     *service.ProductDetail
		mockError      error
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "success with valid product",
			slug: "ethiopian-yirgacheffe",
			mockDetail: &service.ProductDetail{
				Product: repository.Product{
					ID:   mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
					Name: "Ethiopian Yirgacheffe",
					Slug: "ethiopian-yirgacheffe",
					Description: pgtype.Text{
						String: "A bright and floral coffee with notes of bergamot",
						Valid:  true,
					},
					Origin: pgtype.Text{
						String: "Ethiopia",
						Valid:  true,
					},
					RoastLevel: pgtype.Text{
						String: "Light",
						Valid:  true,
					},
				},
				SKUs: []service.ProductSKU{
					{
						SKU: repository.ProductSku{
							ID:                mustParseUUID("223e4567-e89b-12d3-a456-426614174000"),
							Sku:               "ETH-YIR-12OZ-WHOLE",
							WeightValue:       mustParseNumeric("12"),
							WeightUnit:        "oz",
							Grind:             "Whole Bean",
							InventoryQuantity: 100,
						},
						PriceCents:       1599,
						InventoryMessage: "In stock",
					},
					{
						SKU: repository.ProductSku{
							ID:                mustParseUUID("223e4567-e89b-12d3-a456-426614174001"),
							Sku:               "ETH-YIR-12OZ-GROUND",
							WeightValue:       mustParseNumeric("12"),
							WeightUnit:        "oz",
							Grind:             "Ground",
							InventoryQuantity: 5,
							LowStockThreshold: pgtype.Int4{
								Int32: 10,
								Valid: true,
							},
						},
						PriceCents:       1599,
						InventoryMessage: "Low stock",
					},
				},
				Images: []repository.ProductImage{},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Ethiopian Yirgacheffe") {
					t.Error("expected body to contain product name")
				}
				if !strings.Contains(body, "Ethiopia") {
					t.Error("expected body to contain origin")
				}
				if !strings.Contains(body, "Light") {
					t.Error("expected body to contain roast level")
				}
				if !strings.Contains(body, "bright and floral coffee") {
					t.Error("expected body to contain description")
				}
				if !strings.Contains(body, "Available Options") {
					t.Error("expected body to contain SKU section header")
				}
				if !strings.Contains(body, "12 oz") {
					t.Error("expected body to contain weight")
				}
				if !strings.Contains(body, "Whole Bean") {
					t.Error("expected body to contain grind option")
				}
				if !strings.Contains(body, "$15.99") {
					t.Error("expected body to contain formatted price")
				}
				if !strings.Contains(body, "In stock") {
					t.Error("expected body to contain inventory message")
				}
				if !strings.Contains(body, "Low stock") {
					t.Error("expected body to contain low stock message")
				}
			},
		},
		{
			name:           "product not found returns 404",
			slug:           "non-existent-product",
			mockDetail:     nil,
			mockError:      service.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty slug returns 404",
			slug:           "",
			mockDetail:     nil,
			mockError:      nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "service error returns 500",
			slug:           "valid-slug",
			mockDetail:     nil,
			mockError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Failed to load product") {
					t.Error("expected body to contain error message")
				}
			},
		},
		{
			name: "product with no SKUs displays correctly",
			slug: "out-of-production",
			mockDetail: &service.ProductDetail{
				Product: repository.Product{
					ID:   mustParseUUID("123e4567-e89b-12d3-a456-426614174003"),
					Name: "Out of Production Coffee",
					Slug: "out-of-production",
					Description: pgtype.Text{
						String: "No longer available",
						Valid:  true,
					},
					Origin: pgtype.Text{
						String: "Unknown",
						Valid:  true,
					},
					RoastLevel: pgtype.Text{
						String: "Medium",
						Valid:  true,
					},
				},
				SKUs:   []service.ProductSKU{},
				Images: []repository.ProductImage{},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Out of Production Coffee") {
					t.Error("expected body to contain product name")
				}
				if !strings.Contains(body, "Available Options") {
					t.Error("expected body to contain SKU section even if empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				getProductDetailFunc: func(ctx context.Context, slug string) (*service.ProductDetail, error) {
					if slug != tt.slug {
						t.Errorf("expected slug %q, got %q", tt.slug, slug)
					}
					return tt.mockDetail, tt.mockError
				},
			}

			handler := NewProductDetailHandler(mock, nil)

			// Create request with path parameter using pattern matching
			req := httptest.NewRequest(http.MethodGet, "/products/"+tt.slug, nil)
			req.SetPathValue("slug", tt.slug)
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

