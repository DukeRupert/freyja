package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestTenantContext(t *testing.T) {
	t.Run("TenantFromContext returns nil when no tenant", func(t *testing.T) {
		ctx := context.Background()
		tenant := TenantFromContext(ctx)
		if tenant != nil {
			t.Errorf("expected nil tenant, got %+v", tenant)
		}
	})

	t.Run("TenantFromContext returns tenant when set", func(t *testing.T) {
		ctx := context.Background()
		expected := &Tenant{
			ID:     uuid.New(),
			Slug:   "test-roastery",
			Name:   "Test Roastery",
			Status: "active",
		}
		ctx = NewContextWithTenant(ctx, expected)

		tenant := TenantFromContext(ctx)
		if tenant == nil {
			t.Fatal("expected tenant, got nil")
		}
		if tenant.ID != expected.ID {
			t.Errorf("expected ID %v, got %v", expected.ID, tenant.ID)
		}
		if tenant.Slug != expected.Slug {
			t.Errorf("expected Slug %q, got %q", expected.Slug, tenant.Slug)
		}
	})

	t.Run("TenantIDFromContext returns uuid.Nil when no tenant", func(t *testing.T) {
		ctx := context.Background()
		id := TenantIDFromContext(ctx)
		if id != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", id)
		}
	})

	t.Run("TenantIDFromContext returns ID when tenant set", func(t *testing.T) {
		ctx := context.Background()
		expected := &Tenant{ID: uuid.New()}
		ctx = NewContextWithTenant(ctx, expected)

		id := TenantIDFromContext(ctx)
		if id != expected.ID {
			t.Errorf("expected %v, got %v", expected.ID, id)
		}
	})

	t.Run("RequireTenantID panics when no tenant", func(t *testing.T) {
		ctx := context.Background()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		RequireTenantID(ctx)
	})

	t.Run("RequireTenantID returns ID when tenant set", func(t *testing.T) {
		ctx := context.Background()
		expected := &Tenant{ID: uuid.New()}
		ctx = NewContextWithTenant(ctx, expected)

		id := RequireTenantID(ctx)
		if id != expected.ID {
			t.Errorf("expected %v, got %v", expected.ID, id)
		}
	})

	t.Run("MustTenant panics when no tenant", func(t *testing.T) {
		ctx := context.Background()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		MustTenant(ctx)
	})

	t.Run("MustTenant returns tenant when set", func(t *testing.T) {
		ctx := context.Background()
		expected := &Tenant{ID: uuid.New(), Slug: "test"}
		ctx = NewContextWithTenant(ctx, expected)

		tenant := MustTenant(ctx)
		if tenant.ID != expected.ID {
			t.Errorf("expected %v, got %v", expected.ID, tenant.ID)
		}
	})

	t.Run("HasTenant returns false when no tenant", func(t *testing.T) {
		ctx := context.Background()
		if HasTenant(ctx) {
			t.Error("expected HasTenant to return false")
		}
	})

	t.Run("HasTenant returns true when tenant set", func(t *testing.T) {
		ctx := context.Background()
		ctx = NewContextWithTenant(ctx, &Tenant{ID: uuid.New()})
		if !HasTenant(ctx) {
			t.Error("expected HasTenant to return true")
		}
	})
}

func TestUserContext(t *testing.T) {
	t.Run("UserFromContext returns nil when no user", func(t *testing.T) {
		ctx := context.Background()
		user := UserFromContext(ctx)
		if user != nil {
			t.Errorf("expected nil user, got %+v", user)
		}
	})

	t.Run("UserFromContext returns user when set", func(t *testing.T) {
		ctx := context.Background()
		expected := &User{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			Email:       "test@example.com",
			AccountType: "customer",
		}
		ctx = NewContextWithUser(ctx, expected)

		user := UserFromContext(ctx)
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if user.ID != expected.ID {
			t.Errorf("expected ID %v, got %v", expected.ID, user.ID)
		}
		if user.Email != expected.Email {
			t.Errorf("expected Email %q, got %q", expected.Email, user.Email)
		}
	})

	t.Run("UserIDFromContext returns uuid.Nil when no user", func(t *testing.T) {
		ctx := context.Background()
		id := UserIDFromContext(ctx)
		if id != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", id)
		}
	})

	t.Run("RequireUserID panics when no user", func(t *testing.T) {
		ctx := context.Background()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		RequireUserID(ctx)
	})

	t.Run("MustUser panics when no user", func(t *testing.T) {
		ctx := context.Background()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		MustUser(ctx)
	})

	t.Run("IsAuthenticated returns false when no user", func(t *testing.T) {
		ctx := context.Background()
		if IsAuthenticated(ctx) {
			t.Error("expected IsAuthenticated to return false")
		}
	})

	t.Run("IsAuthenticated returns true when user set", func(t *testing.T) {
		ctx := context.Background()
		ctx = NewContextWithUser(ctx, &User{ID: uuid.New()})
		if !IsAuthenticated(ctx) {
			t.Error("expected IsAuthenticated to return true")
		}
	})
}

func TestOperatorContext(t *testing.T) {
	t.Run("OperatorFromContext returns nil when no operator", func(t *testing.T) {
		ctx := context.Background()
		operator := OperatorFromContext(ctx)
		if operator != nil {
			t.Errorf("expected nil operator, got %+v", operator)
		}
	})

	t.Run("OperatorFromContext returns operator when set", func(t *testing.T) {
		ctx := context.Background()
		expected := &Operator{
			ID:       uuid.New(),
			TenantID: uuid.New(),
			Email:    "admin@roastery.com",
			Role:     "owner",
			Status:   "active",
		}
		ctx = NewContextWithOperator(ctx, expected)

		operator := OperatorFromContext(ctx)
		if operator == nil {
			t.Fatal("expected operator, got nil")
		}
		if operator.ID != expected.ID {
			t.Errorf("expected ID %v, got %v", expected.ID, operator.ID)
		}
		if operator.Role != expected.Role {
			t.Errorf("expected Role %q, got %q", expected.Role, operator.Role)
		}
	})

	t.Run("OperatorIDFromContext returns uuid.Nil when no operator", func(t *testing.T) {
		ctx := context.Background()
		id := OperatorIDFromContext(ctx)
		if id != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", id)
		}
	})

	t.Run("RequireOperatorID panics when no operator", func(t *testing.T) {
		ctx := context.Background()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		RequireOperatorID(ctx)
	})

	t.Run("MustOperator panics when no operator", func(t *testing.T) {
		ctx := context.Background()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		MustOperator(ctx)
	})

	t.Run("IsOperator returns false when no operator", func(t *testing.T) {
		ctx := context.Background()
		if IsOperator(ctx) {
			t.Error("expected IsOperator to return false")
		}
	})

	t.Run("IsOperator returns true when operator set", func(t *testing.T) {
		ctx := context.Background()
		ctx = NewContextWithOperator(ctx, &Operator{ID: uuid.New()})
		if !IsOperator(ctx) {
			t.Error("expected IsOperator to return true")
		}
	})

	t.Run("IsOwner returns false when no operator", func(t *testing.T) {
		ctx := context.Background()
		if IsOwner(ctx) {
			t.Error("expected IsOwner to return false")
		}
	})

	t.Run("IsOwner returns false for non-owner role", func(t *testing.T) {
		ctx := context.Background()
		ctx = NewContextWithOperator(ctx, &Operator{ID: uuid.New(), Role: "staff"})
		if IsOwner(ctx) {
			t.Error("expected IsOwner to return false for staff")
		}
	})

	t.Run("IsOwner returns true for owner role", func(t *testing.T) {
		ctx := context.Background()
		ctx = NewContextWithOperator(ctx, &Operator{ID: uuid.New(), Role: "owner"})
		if !IsOwner(ctx) {
			t.Error("expected IsOwner to return true for owner")
		}
	})
}

func TestRequestIDContext(t *testing.T) {
	t.Run("RequestIDFromContext returns empty string when no request ID", func(t *testing.T) {
		ctx := context.Background()
		requestID := RequestIDFromContext(ctx)
		if requestID != "" {
			t.Errorf("expected empty string, got %q", requestID)
		}
	})

	t.Run("RequestIDFromContext returns request ID when set", func(t *testing.T) {
		ctx := context.Background()
		expected := "req-12345"
		ctx = NewContextWithRequestID(ctx, expected)

		requestID := RequestIDFromContext(ctx)
		if requestID != expected {
			t.Errorf("expected %q, got %q", expected, requestID)
		}
	})
}

func TestMultipleContextValues(t *testing.T) {
	t.Run("multiple values can coexist in context", func(t *testing.T) {
		ctx := context.Background()

		tenant := &Tenant{ID: uuid.New(), Slug: "test-roastery"}
		user := &User{ID: uuid.New(), Email: "user@test.com"}
		operator := &Operator{ID: uuid.New(), Role: "owner"}
		requestID := "req-abc123"

		ctx = NewContextWithTenant(ctx, tenant)
		ctx = NewContextWithUser(ctx, user)
		ctx = NewContextWithOperator(ctx, operator)
		ctx = NewContextWithRequestID(ctx, requestID)

		// All values should be retrievable
		if got := TenantFromContext(ctx); got == nil || got.ID != tenant.ID {
			t.Error("tenant not found or wrong ID")
		}
		if got := UserFromContext(ctx); got == nil || got.ID != user.ID {
			t.Error("user not found or wrong ID")
		}
		if got := OperatorFromContext(ctx); got == nil || got.ID != operator.ID {
			t.Error("operator not found or wrong ID")
		}
		if got := RequestIDFromContext(ctx); got != requestID {
			t.Errorf("expected request ID %q, got %q", requestID, got)
		}
	})
}
