package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// PriceListHandler handles all price list related admin routes
type PriceListHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
}

// NewPriceListHandler creates a new price list handler
func NewPriceListHandler(repo repository.Querier, renderer *handler.Renderer) *PriceListHandler {
	return &PriceListHandler{
		repo:     repo,
		renderer: renderer,
	}
}

// List handles GET /admin/price-lists
func (h *PriceListHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	priceLists, err := h.repo.ListAllPriceLists(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Enhance with entry counts
	type DisplayPriceList struct {
		ID            pgtype.UUID
		Name          string
		Description   pgtype.Text
		ListType      string
		IsActive      bool
		IsDefault     bool
		CreatedAt     pgtype.Timestamptz
		CustomerCount int64
	}

	displayLists := make([]DisplayPriceList, len(priceLists))
	for i, pl := range priceLists {
		displayLists[i] = DisplayPriceList{
			ID:          pl.ID,
			Name:        pl.Name,
			Description: pl.Description,
			ListType:    pl.ListType,
			IsActive:    pl.IsActive,
			IsDefault:   pl.ListType == "default",
			CreatedAt:   pl.CreatedAt,
		}
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"PriceLists":  displayLists,
	}

	h.renderer.RenderHTTP(w, "admin/price_lists", data)
}

// Detail handles GET /admin/price-lists/{id}
func (h *PriceListHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	priceListID := r.PathValue("id")
	if priceListID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Price List ID required"))
		return
	}

	var priceListUUID pgtype.UUID
	if err := priceListUUID.Scan(priceListID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price list ID"))
		return
	}

	priceList, err := h.repo.GetPriceListWithEntryCount(ctx, repository.GetPriceListWithEntryCountParams{
		TenantID: tenantID,
		ID:       priceListUUID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	entries, err := h.repo.ListPriceListEntries(ctx, priceListUUID)
	if err != nil {
		entries = nil
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"PriceList":   priceList,
		"Entries":     entries,
	}

	h.renderer.RenderHTTP(w, "admin/price_list_detail", data)
}

// ShowForm handles GET /admin/price-lists/new and GET /admin/price-lists/{id}/edit
func (h *PriceListHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	priceListID := r.PathValue("id")
	var priceList repository.PriceList

	if priceListID != "" {
		var priceListUUID pgtype.UUID
		if err := priceListUUID.Scan(priceListID); err != nil {
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price list ID"))
			return
		}

		pl, err := h.repo.GetPriceListByID(ctx, priceListUUID)
		if err != nil {
			handler.NotFoundResponse(w, r)
			return
		}
		priceList = pl
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"PriceList":   priceList,
		"ListTypes":   []string{"default", "wholesale", "custom"},
	}

	h.renderer.RenderHTTP(w, "admin/price_list_form", data)
}

// HandleForm handles POST /admin/price-lists/new and POST /admin/price-lists/{id}/edit
func (h *PriceListHandler) HandleForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	listType := strings.TrimSpace(r.FormValue("list_type"))
	isActive := r.FormValue("is_active") == "on"

	if name == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Name is required"))
		return
	}

	if listType == "" {
		listType = "custom"
	}

	priceListID := r.PathValue("id")

	if priceListID != "" {
		// Update existing
		var priceListUUID pgtype.UUID
		if err := priceListUUID.Scan(priceListID); err != nil {
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price list ID"))
			return
		}

		_, err := h.repo.UpdatePriceList(ctx, repository.UpdatePriceListParams{
			TenantID:    tenantID,
			ID:          priceListUUID,
			Name:        name,
			Description: pgtype.Text{String: description, Valid: description != ""},
			IsActive:    isActive,
		})
		if err != nil {
			handler.InternalErrorResponse(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/price-lists/"+priceListID, http.StatusSeeOther)
	} else {
		// Create new
		pl, err := h.repo.CreatePriceList(ctx, repository.CreatePriceListParams{
			TenantID:    tenantID,
			Name:        name,
			Description: pgtype.Text{String: description, Valid: description != ""},
			ListType:    listType,
			IsActive:    isActive,
		})
		if err != nil {
			handler.InternalErrorResponse(w, r, err)
			return
		}

		// Format UUID for redirect
		idStr := fmt.Sprintf("%x-%x-%x-%x-%x",
			pl.ID.Bytes[0:4], pl.ID.Bytes[4:6], pl.ID.Bytes[6:8],
			pl.ID.Bytes[8:10], pl.ID.Bytes[10:16])
		http.Redirect(w, r, "/admin/price-lists/"+idStr, http.StatusSeeOther)
	}
}

// UpdateEntry handles POST /admin/price-lists/{id}/entries
func (h *PriceListHandler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	priceListID := r.PathValue("id")
	if priceListID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Price List ID required"))
		return
	}

	var priceListUUID pgtype.UUID
	if err := priceListUUID.Scan(priceListID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price list ID"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	skuID := r.FormValue("sku_id")
	priceStr := r.FormValue("price_cents")
	isAvailable := r.FormValue("is_available") == "on"

	var skuUUID pgtype.UUID
	if err := skuUUID.Scan(skuID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid SKU ID"))
		return
	}

	priceCents, err := strconv.Atoi(priceStr)
	if err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price"))
		return
	}

	err = h.repo.UpsertPriceListEntry(ctx, repository.UpsertPriceListEntryParams{
		TenantID:            tenantID,
		PriceListID:         priceListUUID,
		ProductSkuID:        skuUUID,
		PriceCents:          int32(priceCents),
		CompareAtPriceCents: pgtype.Int4{},
		IsAvailable:         isAvailable,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Price updated"))
		return
	}

	http.Redirect(w, r, "/admin/price-lists/"+priceListID, http.StatusSeeOther)
}

// Delete handles POST /admin/price-lists/{id}/delete
func (h *PriceListHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	priceListID := r.PathValue("id")
	if priceListID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Price List ID required"))
		return
	}

	var priceListUUID pgtype.UUID
	if err := priceListUUID.Scan(priceListID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price list ID"))
		return
	}

	// Check it's not the default
	pl, err := h.repo.GetPriceListByID(ctx, priceListUUID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if pl.ListType == "default" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EFORBIDDEN, "", "Cannot delete the default price list"))
		return
	}

	err = h.repo.DeletePriceList(ctx, repository.DeletePriceListParams{
		TenantID: tenantID,
		ID:       priceListUUID,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/price-lists", http.StatusSeeOther)
}
