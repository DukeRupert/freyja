package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductFormHandler handles both new product and edit product pages
type ProductFormHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewProductFormHandler creates a new product form handler
func NewProductFormHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *ProductFormHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProductFormHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *ProductFormHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.showForm(w, r)
	} else if r.Method == http.MethodPost {
		h.handleSubmit(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProductFormHandler) showForm(w http.ResponseWriter, r *http.Request) {
	// Check if this is edit (has product ID in path)
	productID := r.PathValue("id")

	var product repository.Product
	var tastingNotesString string

	if productID != "" {
		// Edit mode - load existing product
		var productUUID pgtype.UUID
		if err := productUUID.Scan(productID); err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		p, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
			TenantID: h.tenantID,
			ID:       productUUID,
		})
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		product = p

		// Convert tasting notes array to comma-separated string
		if len(product.TastingNotes) > 0 {
			tastingNotesString = strings.Join(product.TastingNotes, ", ")
		}
	} else {
		// New mode - set defaults
		product.Status = "draft"
		product.Visibility = "public"
	}

	data := map[string]interface{}{
		"CurrentPath":        r.URL.Path,
		"Product":            product,
		"TastingNotesString": tastingNotesString,
	}

	h.renderer.RenderHTTP(w, "admin/product_form", data)
}

func (h *ProductFormHandler) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Check if this is edit or create
	productID := r.PathValue("id")
	isEdit := productID != ""

	// Parse tasting notes from comma-separated string
	tastingNotesStr := strings.TrimSpace(r.FormValue("tasting_notes"))
	var tastingNotes []string
	if tastingNotesStr != "" {
		parts := strings.Split(tastingNotesStr, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				tastingNotes = append(tastingNotes, trimmed)
			}
		}
	}

	if isEdit {
		// Update existing product
		var productUUID pgtype.UUID
		if err := productUUID.Scan(productID); err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		elevationMin := pgtype.Int4{}
		if minStr := r.FormValue("elevation_min"); minStr != "" {
			if min, err := strconv.Atoi(minStr); err == nil {
				elevationMin.Int32 = int32(min)
				elevationMin.Valid = true
			}
		}

		elevationMax := pgtype.Int4{}
		if maxStr := r.FormValue("elevation_max"); maxStr != "" {
			if max, err := strconv.Atoi(maxStr); err == nil {
				elevationMax.Int32 = int32(max)
				elevationMax.Valid = true
			}
		}

		sortOrder := int32(0)
		if sortStr := r.FormValue("sort_order"); sortStr != "" {
			if sort, err := strconv.Atoi(sortStr); err == nil {
				sortOrder = int32(sort)
			}
		}

		_, err := h.repo.UpdateProduct(r.Context(), repository.UpdateProductParams{
			TenantID:         h.tenantID,
			ID:               productUUID,
			Name:             r.FormValue("name"),
			Slug:             r.FormValue("slug"),
			ShortDescription: makePgText(r.FormValue("short_description")),
			Description:      makePgText(r.FormValue("description")),
			Status:           r.FormValue("status"),
			Visibility:       r.FormValue("visibility"),
			Origin:           makePgText(r.FormValue("origin")),
			Region:           makePgText(r.FormValue("region")),
			Producer:         makePgText(r.FormValue("producer")),
			Process:          makePgText(r.FormValue("process")),
			RoastLevel:       makePgText(r.FormValue("roast_level")),
			TastingNotes:     tastingNotes,
			ElevationMin:     elevationMin,
			ElevationMax:     elevationMax,
			SortOrder:        sortOrder,
		})
		if err != nil {
			http.Error(w, "Failed to update product: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to product list
		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
	} else {
		// Create new product
		elevationMin := pgtype.Int4{}
		if minStr := r.FormValue("elevation_min"); minStr != "" {
			if min, err := strconv.Atoi(minStr); err == nil {
				elevationMin.Int32 = int32(min)
				elevationMin.Valid = true
			}
		}

		elevationMax := pgtype.Int4{}
		if maxStr := r.FormValue("elevation_max"); maxStr != "" {
			if max, err := strconv.Atoi(maxStr); err == nil {
				elevationMax.Int32 = int32(max)
				elevationMax.Valid = true
			}
		}

		sortOrder := int32(0)
		if sortStr := r.FormValue("sort_order"); sortStr != "" {
			if sort, err := strconv.Atoi(sortStr); err == nil {
				sortOrder = int32(sort)
			}
		}

		// Set white label fields to NULL/false for now
		isWhiteLabel := false
		baseProductID := pgtype.UUID{Valid: false}
		whiteLabelCustomerID := pgtype.UUID{Valid: false}

		_, err := h.repo.CreateProduct(r.Context(), repository.CreateProductParams{
			TenantID:             h.tenantID,
			Name:                 r.FormValue("name"),
			Slug:                 r.FormValue("slug"),
			ShortDescription:     makePgText(r.FormValue("short_description")),
			Description:          makePgText(r.FormValue("description")),
			Status:               r.FormValue("status"),
			Visibility:           r.FormValue("visibility"),
			Origin:               makePgText(r.FormValue("origin")),
			Region:               makePgText(r.FormValue("region")),
			Producer:             makePgText(r.FormValue("producer")),
			Process:              makePgText(r.FormValue("process")),
			RoastLevel:           makePgText(r.FormValue("roast_level")),
			TastingNotes:         tastingNotes,
			ElevationMin:         elevationMin,
			ElevationMax:         elevationMax,
			IsWhiteLabel:         isWhiteLabel,
			BaseProductID:        baseProductID,
			WhiteLabelCustomerID: whiteLabelCustomerID,
			SortOrder:            sortOrder,
		})
		if err != nil {
			http.Error(w, "Failed to create product: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to product list
		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
	}
}

// makePgText creates a pgtype.Text from a string
func makePgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
