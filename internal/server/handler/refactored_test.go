package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/dukerupert/freyja/internal/database"
	h "github.com/dukerupert/freyja/internal/server/handler"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock event publisher for testing
type mockEventPublisher struct{}

func (m *mockEventPublisher) Publish(topic string, event interface{}) error {
	return nil
}

func (m *mockEventPublisher) Close() error {
	return nil
}

// Custom validator for echo
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Test setup helper
func setupTest(t *testing.T) (*echo.Echo, *database.DB, zerolog.Logger) {
	// Setup database connection (use test database URL)
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}

	db, err := database.NewDB(dbURL)
	require.NoError(t, err)

	// Setup logger (disabled for tests)
	logger := zerolog.Nop()

	// Setup Echo instance
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	return e, db, logger
}

// Cleanup helper
func cleanupTest(t *testing.T, db *database.DB) {
	// Clean up test data
	ctx := context.Background()
	
	// Delete test data in reverse dependency order
	_, _ = db.Pool.Exec(ctx, "DELETE FROM product_option_values WHERE value LIKE 'Test%'")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM product_options WHERE option_key LIKE 'test%'")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM products WHERE name LIKE 'Test%'")
	
	db.Close()
}

func TestProductCRUD(t *testing.T) {
	e, db, logger := setupTest(t)
	defer cleanupTest(t, db)

	eventBus := &mockEventPublisher{}

	t.Run("Complete Product CRUD Flow", func(t *testing.T) {
		var productID int32

		// Test Create Product
		t.Run("Create Product", func(t *testing.T) {
			productData := map[string]interface{}{
				"name":        "Test Coffee Blend",
				"description": "A delicious test coffee blend",
				"active":      true,
			}

			jsonData, _ := json.Marshal(productData)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(jsonData))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := h.HandleCreateProduct(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			// Extract product ID for subsequent tests
			data := response["data"].(map[string]interface{})
			productID = int32(data["id"].(float64))
			assert.Greater(t, productID, int32(0))
		})

		// Test Read Product
		t.Run("Read Product", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+strconv.Itoa(int(productID)), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(productID)))

			handler := h.HandleGetProduct(db, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			product := response["product"].(map[string]interface{})
			assert.Equal(t, "Test Coffee Blend", product["name"])
			assert.Equal(t, "A delicious test coffee blend", product["description"])
			assert.True(t, product["active"].(bool))
		})

		// Test Update Product
		t.Run("Update Product", func(t *testing.T) {
			updateData := map[string]interface{}{
				"name":        "Updated Test Coffee Blend",
				"description": "An updated delicious test coffee blend",
				"active":      true,
			}

			jsonData, _ := json.Marshal(updateData)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+strconv.Itoa(int(productID)), bytes.NewReader(jsonData))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(productID)))

			handler := h.HandleUpdateProduct(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, "Updated Test Coffee Blend", data["name"])
		})

		// Test Option and Option Value operations
		var optionID, optionValueID int32

		// Test Create Product Option
		t.Run("Create Product Option", func(t *testing.T) {
			optionData := map[string]interface{}{
				"option_key": "test_size",
			}

			jsonData, _ := json.Marshal(optionData)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+strconv.Itoa(int(productID))+"/options", bytes.NewReader(jsonData))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(productID)))

			handler := h.HandleCreateProductOption(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			optionID = int32(data["id"].(float64))
			assert.Greater(t, optionID, int32(0))
		})

		// Test Read Product Option
		t.Run("Read Product Option", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+strconv.Itoa(int(productID))+"/options", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(productID)))

			handler := h.HandleGetProductOptions(db, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			options := response["options"].([]interface{})
			assert.Greater(t, len(options), 0)

			option := options[0].(map[string]interface{})
			assert.Equal(t, "test_size", option["option_key"])
		})

		// Test Update Product Option
		t.Run("Update Product Option", func(t *testing.T) {
			updateData := map[string]interface{}{
				"option_key": "test_updated_size",
			}

			jsonData, _ := json.Marshal(updateData)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/options/"+strconv.Itoa(int(optionID)), bytes.NewReader(jsonData))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(optionID)))

			handler := h.HandleUpdateProductOption(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, "test_updated_size", data["option_key"])
		})

		// Test Create Option Value
		t.Run("Create Option Value", func(t *testing.T) {
			valueData := map[string]interface{}{
				"value": "Test Large",
			}

			jsonData, _ := json.Marshal(valueData)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/options/"+strconv.Itoa(int(optionID))+"/values", bytes.NewReader(jsonData))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(optionID)))

			handler := h.HandleCreateProductOptionValue(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			optionValueID = int32(data["id"].(float64))
			assert.Greater(t, optionValueID, int32(0))
			assert.Equal(t, "Test Large", data["value"])
		})

		// Test Read Option Value
		t.Run("Read Option Value", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/option-values/"+strconv.Itoa(int(optionValueID)), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(optionValueID)))

			handler := h.HandleGetProductOptionValue(db, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, "Test Large", data["value"])
		})

		// Test Update Option Value
		t.Run("Update Option Value", func(t *testing.T) {
			updateData := map[string]interface{}{
				"value": "Test Extra Large",
			}

			jsonData, _ := json.Marshal(updateData)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/option-values/"+strconv.Itoa(int(optionValueID)), bytes.NewReader(jsonData))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(optionValueID)))

			handler := h.HandleUpdateProductOptionValue(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, "Test Extra Large", data["value"])
		})

		// Test Delete Option Value
		t.Run("Delete Option Value", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/option-values/"+strconv.Itoa(int(optionValueID)), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(optionValueID)))

			handler := h.HandleDeleteProductOptionValue(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))
			assert.Equal(t, "Option value deleted successfully", response["message"])
		})

		// Test Delete Product Option
		t.Run("Delete Product Option", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/options/"+strconv.Itoa(int(optionID)), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(optionID)))

			handler := h.HandleDeleteProductOption(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))
			assert.Equal(t, "Product option deleted successfully", response["message"])
		})

		// Test Delete Product
		t.Run("Delete Product", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+strconv.Itoa(int(productID)), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(strconv.Itoa(int(productID)))

			handler := h.HandleDeleteProduct(db, eventBus, logger)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))
			assert.Equal(t, "Product deleted successfully", response["message"])
		})
	})
}

func TestErrorHandling(t *testing.T) {
	e, db, logger := setupTest(t)
	defer cleanupTest(t, db)

	eventBus := &mockEventPublisher{}

	t.Run("Invalid Product ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		handler := h.HandleGetProduct(db, logger)
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "INVALID_ID", response["code"])
	})

	t.Run("Product Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/99999", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		handler := h.HandleGetProduct(db, logger)
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "PRODUCT_NOT_FOUND", response["code"])
	})

	t.Run("Invalid JSON in Create Product", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader([]byte("invalid json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := h.HandleCreateProduct(db, eventBus, logger)
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "INVALID_JSON", response["code"])
	})

	t.Run("Missing Required Fields", func(t *testing.T) {
		productData := map[string]interface{}{
			"description": "Missing name field",
		}

		jsonData, _ := json.Marshal(productData)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(jsonData))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := h.HandleCreateProduct(db, eventBus, logger)
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", response["code"])
	})
}

func TestConcurrentOperations(t *testing.T) {
	e, db, logger := setupTest(t)
	defer cleanupTest(t, db)

	eventBus := &mockEventPublisher{}

	// Create a product for concurrent testing
	productData := map[string]interface{}{
		"name":        "Concurrent Test Coffee",
		"description": "For testing concurrent operations",
		"active":      true,
	}

	jsonData, _ := json.Marshal(productData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(jsonData))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := h.HandleCreateProduct(db, eventBus, logger)
	err := handler(c)
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	productID := int32(data["id"].(float64))

	t.Run("Concurrent Read Operations", func(t *testing.T) {
		// Test that multiple concurrent read operations work correctly
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+strconv.Itoa(int(productID)), nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("id")
				c.SetParamValues(strconv.Itoa(int(productID)))

				handler := h.HandleGetProduct(db, logger)
				err := handler(c)

				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}