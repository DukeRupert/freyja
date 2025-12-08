package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "message only",
			err: &Error{
				Code:    EINVALID,
				Message: "invalid input",
			},
			expected: "invalid input",
		},
		{
			name: "with operation",
			err: &Error{
				Code:    EINVALID,
				Op:      "product.create",
				Message: "invalid input",
			},
			expected: "product.create: invalid input",
		},
		{
			name: "with wrapped error",
			err: &Error{
				Code:    EINTERNAL,
				Op:      "product.create",
				Message: "failed to save",
				Err:     errors.New("database connection failed"),
			},
			expected: "product.create: failed to save: database connection failed",
		},
		{
			name: "wrapped error without op",
			err: &Error{
				Code:    EINTERNAL,
				Message: "failed to save",
				Err:     errors.New("database connection failed"),
			},
			expected: "failed to save: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &Error{
		Code:    EINTERNAL,
		Message: "wrapped",
		Err:     underlying,
	}

	if unwrapped := err.Unwrap(); unwrapped != underlying {
		t.Errorf("Error.Unwrap() = %v, want %v", unwrapped, underlying)
	}

	// Test errors.Is works through unwrapping
	if !errors.Is(err, underlying) {
		t.Error("errors.Is should find underlying error")
	}
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "domain error",
			err:      &Error{Code: EINVALID, Message: "test"},
			expected: EINVALID,
		},
		{
			name:     "wrapped domain error",
			err:      fmt.Errorf("wrapped: %w", &Error{Code: ENOTFOUND, Message: "test"}),
			expected: ENOTFOUND,
		},
		{
			name:     "non-domain error",
			err:      errors.New("some error"),
			expected: EINTERNAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorCode(tt.err); got != tt.expected {
				t.Errorf("ErrorCode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "domain error with message",
			err:      &Error{Code: EINVALID, Message: "invalid product name"},
			expected: "invalid product name",
		},
		{
			name:     "internal error hides message",
			err:      &Error{Code: EINTERNAL, Message: "database connection string leaked"},
			expected: "An internal error occurred. Please try again later.",
		},
		{
			name:     "non-domain error returns generic message",
			err:      errors.New("some internal detail"),
			expected: "An internal error occurred. Please try again later.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorMessage(tt.err); got != tt.expected {
				t.Errorf("ErrorMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorOp(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "domain error with op",
			err:      &Error{Code: EINVALID, Op: "product.create", Message: "test"},
			expected: "product.create",
		},
		{
			name:     "domain error without op",
			err:      &Error{Code: EINVALID, Message: "test"},
			expected: "",
		},
		{
			name:     "non-domain error",
			err:      errors.New("test"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorOp(tt.err); got != tt.expected {
				t.Errorf("ErrorOp() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf(EINVALID, "product.validate", "invalid price: %d", -100)

	var domainErr *Error
	if !errors.As(err, &domainErr) {
		t.Fatal("Errorf should return *Error")
	}

	if domainErr.Code != EINVALID {
		t.Errorf("Code = %q, want %q", domainErr.Code, EINVALID)
	}

	if domainErr.Op != "product.validate" {
		t.Errorf("Op = %q, want %q", domainErr.Op, "product.validate")
	}

	if domainErr.Message != "invalid price: -100" {
		t.Errorf("Message = %q, want %q", domainErr.Message, "invalid price: -100")
	}
}

func TestWrapError(t *testing.T) {
	t.Run("wraps non-nil error", func(t *testing.T) {
		underlying := errors.New("db error")
		err := WrapError(underlying, EINTERNAL, "product.save", "failed to save product")

		var domainErr *Error
		if !errors.As(err, &domainErr) {
			t.Fatal("WrapError should return *Error")
		}

		if domainErr.Code != EINTERNAL {
			t.Errorf("Code = %q, want %q", domainErr.Code, EINTERNAL)
		}

		if !errors.Is(err, underlying) {
			t.Error("should wrap underlying error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapError(nil, EINTERNAL, "test", "test")
		if err != nil {
			t.Errorf("WrapError(nil) should return nil, got %v", err)
		}
	})
}

func TestIsCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{
			name:     "matching code",
			err:      &Error{Code: ENOTFOUND, Message: "test"},
			code:     ENOTFOUND,
			expected: true,
		},
		{
			name:     "non-matching code",
			err:      &Error{Code: EINVALID, Message: "test"},
			code:     ENOTFOUND,
			expected: false,
		},
		{
			name:     "non-domain error matches EINTERNAL",
			err:      errors.New("test"),
			code:     EINTERNAL,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCode(tt.err, tt.code); got != tt.expected {
				t.Errorf("IsCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	t.Run("single field error", func(t *testing.T) {
		err := NewValidationError("product.create", "name", "name is required")

		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatal("NewValidationError should return *ValidationError")
		}

		if ve.Op != "product.create" {
			t.Errorf("Op = %q, want %q", ve.Op, "product.create")
		}

		if msg, ok := ve.Fields["name"]; !ok || msg != "name is required" {
			t.Errorf("Fields[name] = %q, want %q", msg, "name is required")
		}

		expected := "product.create: name: name is required"
		if ve.Error() != expected {
			t.Errorf("Error() = %q, want %q", ve.Error(), expected)
		}
	})

	t.Run("multiple field errors", func(t *testing.T) {
		err := NewValidationError("product.create", "name", "name is required")
		err = AddFieldError(err, "price", "price must be positive")

		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatal("should be ValidationError")
		}

		if len(ve.Fields) != 2 {
			t.Errorf("Fields count = %d, want 2", len(ve.Fields))
		}
	})

	t.Run("add field to non-validation error", func(t *testing.T) {
		err := AddFieldError(nil, "name", "name is required")

		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatal("AddFieldError(nil) should return *ValidationError")
		}

		if len(ve.Fields) != 1 {
			t.Errorf("Fields count = %d, want 1", len(ve.Fields))
		}
	})
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error",
			err:      NewValidationError("test", "field", "error"),
			expected: true,
		},
		{
			name:     "domain error",
			err:      &Error{Code: EINVALID, Message: "test"},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("test"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidationError(tt.err); got != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetValidationFields(t *testing.T) {
	t.Run("validation error", func(t *testing.T) {
		err := NewValidationError("test", "name", "required")
		fields := GetValidationFields(err)

		if fields == nil {
			t.Fatal("GetValidationFields should return fields map")
		}

		if fields["name"] != "required" {
			t.Errorf("fields[name] = %q, want %q", fields["name"], "required")
		}
	})

	t.Run("non-validation error", func(t *testing.T) {
		err := errors.New("test")
		fields := GetValidationFields(err)

		if fields != nil {
			t.Errorf("GetValidationFields should return nil for non-validation error")
		}
	})
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		err := NotFound("product.get", "product", "abc-123")
		if ErrorCode(err) != ENOTFOUND {
			t.Errorf("NotFound code = %q, want %q", ErrorCode(err), ENOTFOUND)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		err := Unauthorized("auth.check", "invalid credentials")
		if ErrorCode(err) != EUNAUTHORIZED {
			t.Errorf("Unauthorized code = %q, want %q", ErrorCode(err), EUNAUTHORIZED)
		}
	})

	t.Run("Forbidden", func(t *testing.T) {
		err := Forbidden("product.delete", "only owner can delete")
		if ErrorCode(err) != EFORBIDDEN {
			t.Errorf("Forbidden code = %q, want %q", ErrorCode(err), EFORBIDDEN)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		err := Invalid("product.create", "price must be positive")
		if ErrorCode(err) != EINVALID {
			t.Errorf("Invalid code = %q, want %q", ErrorCode(err), EINVALID)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		err := Conflict("product.create", "slug already exists")
		if ErrorCode(err) != ECONFLICT {
			t.Errorf("Conflict code = %q, want %q", ErrorCode(err), ECONFLICT)
		}
	})

	t.Run("Internal", func(t *testing.T) {
		underlying := errors.New("db error")
		err := Internal(underlying, "product.save", "failed to save")

		if ErrorCode(err) != EINTERNAL {
			t.Errorf("Internal code = %q, want %q", ErrorCode(err), EINTERNAL)
		}

		if !errors.Is(err, underlying) {
			t.Error("Internal should wrap underlying error")
		}

		// Message should be hidden
		msg := ErrorMessage(err)
		if msg != "An internal error occurred. Please try again later." {
			t.Errorf("Internal message should be hidden, got %q", msg)
		}
	})
}

func TestPreDefinedErrors(t *testing.T) {
	t.Run("ErrTenantMismatch", func(t *testing.T) {
		if ErrorCode(ErrTenantMismatch) != EFORBIDDEN {
			t.Errorf("ErrTenantMismatch code = %q, want %q", ErrorCode(ErrTenantMismatch), EFORBIDDEN)
		}
	})

	t.Run("ErrTenantRequired", func(t *testing.T) {
		if ErrorCode(ErrTenantRequired) != EINTERNAL {
			t.Errorf("ErrTenantRequired code = %q, want %q", ErrorCode(ErrTenantRequired), EINTERNAL)
		}
	})
}
