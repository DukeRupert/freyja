package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// CustomerListHandler handles the admin customer list page
type CustomerListHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewCustomerListHandler creates a new customer list handler
func NewCustomerListHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *CustomerListHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &CustomerListHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *CustomerListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filter from query params (optional)
	accountType := r.URL.Query().Get("type")

	var users []repository.User
	var err error

	if accountType != "" {
		// Filter by account type
		users, err = h.repo.ListUsersByAccountType(r.Context(), repository.ListUsersByAccountTypeParams{
			TenantID:    h.tenantID,
			AccountType: accountType,
		})
	} else {
		// Get all users
		users, err = h.repo.ListUsers(r.Context(), repository.ListUsersParams{
			TenantID: h.tenantID,
			Limit:    100,
			Offset:   0,
		})
	}

	if err != nil {
		http.Error(w, "Failed to load customers", http.StatusInternalServerError)
		return
	}

	// Format users for display
	type DisplayUser struct {
		ID                  pgtype.UUID
		Email               string
		FirstName           pgtype.Text
		LastName            pgtype.Text
		FullName            string
		AccountType         string
		Status              string
		CreatedAt           pgtype.Timestamptz
		CreatedAtFormatted  string
		LastLoginFormatted  string
		CompanyName         pgtype.Text
		WholesaleStatus     pgtype.Text
	}

	displayUsers := make([]DisplayUser, len(users))
	for i, user := range users {
		createdAtFormatted := ""
		if user.CreatedAt.Valid {
			createdAtFormatted = user.CreatedAt.Time.Format("Jan 2, 2006")
		}

		fullName := ""
		if user.FirstName.Valid && user.LastName.Valid {
			fullName = user.FirstName.String + " " + user.LastName.String
		} else if user.FirstName.Valid {
			fullName = user.FirstName.String
		} else if user.LastName.Valid {
			fullName = user.LastName.String
		}

		displayUsers[i] = DisplayUser{
			ID:                 user.ID,
			Email:              user.Email,
			FirstName:          user.FirstName,
			LastName:           user.LastName,
			FullName:           fullName,
			AccountType:        user.AccountType,
			Status:             user.Status,
			CreatedAt:          user.CreatedAt,
			CreatedAtFormatted: createdAtFormatted,
			LastLoginFormatted: "Never", // TODO: Add last_login_at to users table
			CompanyName:        user.CompanyName,
			WholesaleStatus:    user.WholesaleApplicationStatus,
		}
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Customers":   displayUsers,
		"FilterType":  accountType,
	}

	h.renderer.RenderHTTP(w, "admin/customers", data)
}
