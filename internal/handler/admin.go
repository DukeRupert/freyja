// internal/handler/admin.go
package handler

import (
	"net/http"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	adminService interfaces.AdminService
}

func NewAdminHandler(adminService interfaces.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// BackfillCustomers handles POST /api/v1/admin/backfill/customers
func (h *AdminHandler) BackfillCustomers(c echo.Context) error {
	ctx := c.Request().Context()

	var req interfaces.BackfillCustomersRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	result, err := h.adminService.BackfillCustomerStripeSync(ctx, req)
	if err != nil {
		c.Logger().Errorf("Failed to start customer backfill: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to start customer backfill",
			"code":  "BACKFILL_FAILED",
		})
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message": "Customer backfill started",
		"job":     result,
	})
}

// BackfillProducts handles POST /api/v1/admin/backfill/products
func (h *AdminHandler) BackfillProducts(c echo.Context) error {
	ctx := c.Request().Context()

	var req interfaces.BackfillProductsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	result, err := h.adminService.BackfillProductStripeSync(ctx, req)
	if err != nil {
		c.Logger().Errorf("Failed to start product backfill: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to start product backfill",
			"code":  "BACKFILL_FAILED",
		})
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message": "Product backfill started",
		"job":     result,
	})
}

// GetSyncStatus handles GET /api/v1/admin/sync/status
func (h *AdminHandler) GetSyncStatus(c echo.Context) error {
	ctx := c.Request().Context()

	status, err := h.adminService.GetSyncStatus(ctx)
	if err != nil {
		c.Logger().Errorf("Failed to get sync status: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get sync status",
			"code":  "SYNC_STATUS_FAILED",
		})
	}

	return c.JSON(http.StatusOK, status)
}

// GetBackfillStatus handles GET /api/v1/admin/backfill/:job_id/status
func (h *AdminHandler) GetBackfillStatus(c echo.Context) error {
	ctx := c.Request().Context()
	jobID := c.Param("job_id")

	if jobID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Job ID is required",
			"code":  "MISSING_JOB_ID",
		})
	}

	status, err := h.adminService.GetBackfillStatus(ctx, jobID)
	if err != nil {
		c.Logger().Errorf("Failed to get backfill status for job %s: %v", jobID, err)

		if err.Error() == "job not found: "+jobID {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Backfill job not found",
				"code":  "JOB_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get backfill status",
			"code":  "BACKFILL_STATUS_FAILED",
		})
	}

	return c.JSON(http.StatusOK, status)
}
