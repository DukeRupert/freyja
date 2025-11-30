package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductListHandler handles the admin product list page
type ProductListHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewProductListHandler creates a new product list handler
func NewProductListHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *ProductListHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProductListHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *ProductListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all products (including draft and archived)
	products, err := h.repo.ListAllProducts(r.Context(), h.tenantID)
	if err != nil {
		http.Error(w, "Failed to load products", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Products":    products,
	}

	h.renderer.RenderHTTP(w, "admin/products", data)
}
