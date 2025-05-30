// internal/handler/product_handler_test.go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dukerupert/freyja/internal/api"
	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repo"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"github.com/stretchr/testify/suite"
)

// ProductHandlerTestSuite holds the test suite state
type ProductHandlerTestSuite struct {
	suite.Suite
	db             *pgxpool.Pool
	echo           *echo.Echo
	productService *service.ProductService
	cfg            *config.Config
}

// SetupSuite runs once before all tests in the suite
func (suite *ProductHandlerTestSuite) SetupSuite() {
	// Load configuration once
	cfg, err := config.Load("../../.env")
	suite.Require().NoError(err, "Failed to load configuration")
	suite.cfg = cfg

	// Modify config for test database
	suite.cfg.DB.Name = suite.cfg.DB.Name + "_test"
	
	// Create test database connection
	db, err := pgxpool.New(context.Background(), suite.cfg.DB.DSN)
	if err != nil {
		suite.T().Skip("Test database not available")
		return
	}
	suite.db = db

	// Test database connection
	err = db.Ping(context.Background())
	suite.Require().NoError(err, "Failed to ping test database")

	// Setup services
	queries := repo.New(db)
	suite.productService = service.NewProductService(queries)
	productHandler := handler.NewProductHandler(suite.productService)

	// Setup Echo server
	e := echo.New()
	strictHandler := api.NewStrictHandler(productHandler, []strictecho.StrictEchoMiddlewareFunc{})
	api.RegisterHandlersWithBaseURL(e, strictHandler, "/api/v1")
	suite.echo = e
}

// TearDownSuite runs once after all tests in the suite
func (suite *ProductHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest runs before each individual test
func (suite *ProductHandlerTestSuite) SetupTest() {
	// Clean up test data before each test
	if suite.db != nil {
		_, err := suite.db.Exec(context.Background(), "TRUNCATE TABLE products CASCADE")
		suite.Require().NoError(err, "Failed to clean test data")
	}
}

// Helper method to create a test product
func (suite *ProductHandlerTestSuite) createTestProduct(title, handle string) *api.Product {
	createReq := api.CreateProductRequest{
		Title:  title,
		Handle: handle,
	}

	product, err := suite.productService.CreateProduct(context.Background(), createReq)
	suite.Require().NoError(err, "Failed to create test product")
	return product
}

// Helper method to make HTTP requests
func (suite *ProductHandlerTestSuite) makeRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
	var reqBody *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(bodyBytes)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()

	suite.echo.ServeHTTP(rec, req)
	return rec, nil
}

// Test methods
func (suite *ProductHandlerTestSuite) TestCreateProduct() {
	createReq := api.CreateProductRequest{
		Title:  "Test Coffee",
		Handle: "test-coffee",
	}

	rec, err := suite.makeRequest(http.MethodPost, "/api/v1/products", createReq)
	suite.Require().NoError(err)

	suite.Equal(http.StatusCreated, rec.Code)

	var product api.Product
	err = json.Unmarshal(rec.Body.Bytes(), &product)
	suite.Require().NoError(err)

	suite.Equal("Test Coffee", product.Title)
	suite.Equal("test-coffee", product.Handle)
	suite.NotEmpty(product.Id)
}

func (suite *ProductHandlerTestSuite) TestCreateProductWithMetadata() {
	metadata := map[string]interface{}{
		"cupping_score": 87,
		"certifications": []string{"organic", "fair_trade"},
	}

	createReq := api.CreateProductRequest{
		Title:    "Ethiopian Coffee",
		Handle:   "ethiopian-coffee",
		Metadata: &metadata,
	}

	rec, err := suite.makeRequest(http.MethodPost, "/api/v1/products", createReq)
	suite.Require().NoError(err)

	suite.Equal(http.StatusCreated, rec.Code)

	var product api.Product
	err = json.Unmarshal(rec.Body.Bytes(), &product)
	suite.Require().NoError(err)

	suite.Equal("Ethiopian Coffee", product.Title)
	suite.NotNil(product.Metadata)
	
	// Verify metadata was stored correctly
	score, ok := (*product.Metadata)["cupping_score"]
	suite.True(ok)
	suite.Equal(float64(87), score) // JSON numbers are float64
}

func (suite *ProductHandlerTestSuite) TestGetProduct() {
	// Create a product first
	product := suite.createTestProduct("Test Coffee", "test-coffee")

	// Get the product
	rec, err := suite.makeRequest(http.MethodGet, "/api/v1/products/"+product.Id.String(), nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusOK, rec.Code)

	var retrievedProduct api.Product
	err = json.Unmarshal(rec.Body.Bytes(), &retrievedProduct)
	suite.Require().NoError(err)

	suite.Equal(product.Id, retrievedProduct.Id)
	suite.Equal(product.Title, retrievedProduct.Title)
}

func (suite *ProductHandlerTestSuite) TestGetProductNotFound() {
	// Try to get a non-existent product
	fakeUUID := "123e4567-e89b-12d3-a456-426614174000"
	rec, err := suite.makeRequest(http.MethodGet, "/api/v1/products/"+fakeUUID, nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusNotFound, rec.Code)

	var errorResponse api.Error
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	suite.Require().NoError(err)

	suite.Equal("product_not_found", errorResponse.Error)
}

func (suite *ProductHandlerTestSuite) TestListProducts() {
	// Create some test products
	products := []struct {
		title, handle string
	}{
		{"Coffee 1", "coffee-1"},
		{"Coffee 2", "coffee-2"},
		{"Coffee 3", "coffee-3"},
	}

	for _, p := range products {
		suite.createTestProduct(p.title, p.handle)
	}

	// List products
	rec, err := suite.makeRequest(http.MethodGet, "/api/v1/products", nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusOK, rec.Code)

	var response struct {
		Products   []api.Product      `json:"products"`
		Pagination api.PaginationMeta `json:"pagination"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.Len(response.Products, 3)
	suite.Equal(3, response.Pagination.Total)
}

func (suite *ProductHandlerTestSuite) TestListProductsWithPagination() {
	// Create 5 test products
	for i := 1; i <= 5; i++ {
		suite.createTestProduct(
			fmt.Sprintf("Coffee %d", i),
			fmt.Sprintf("coffee-%d", i),
		)
	}

	// Test pagination: page 1, limit 2
	rec, err := suite.makeRequest(http.MethodGet, "/api/v1/products?page=1&limit=2", nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusOK, rec.Code)

	var response struct {
		Products   []api.Product      `json:"products"`
		Pagination api.PaginationMeta `json:"pagination"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.Len(response.Products, 2)
	suite.Equal(1, response.Pagination.Page)
	suite.Equal(2, response.Pagination.Limit)
	suite.Equal(5, response.Pagination.Total)
	suite.Equal(3, response.Pagination.TotalPages) // ceil(5/2)
}

func (suite *ProductHandlerTestSuite) TestSearchProducts() {
	// Create some test products
	products := []struct {
		title, handle string
	}{
		{"Ethiopian Coffee", "ethiopian-coffee"},
		{"Colombian Coffee", "colombian-coffee"},
		{"Brazilian Tea", "brazilian-tea"},
	}

	for _, p := range products {
		suite.createTestProduct(p.title, p.handle)
	}

	// Search for coffee
	rec, err := suite.makeRequest(http.MethodGet, "/api/v1/products/search?q=coffee", nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusOK, rec.Code)

	var response struct {
		Products   []api.Product      `json:"products"`
		Pagination api.PaginationMeta `json:"pagination"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Should find 2 products with "coffee" in the title
	suite.Len(response.Products, 2)
}

func (suite *ProductHandlerTestSuite) TestDeleteProduct() {
	// Create a product first
	product := suite.createTestProduct("Test Coffee", "test-coffee")

	// Delete the product
	rec, err := suite.makeRequest(http.MethodDelete, "/api/v1/products/"+product.Id.String(), nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusNoContent, rec.Code)

	// Try to get the deleted product (should return 404)
	rec, err = suite.makeRequest(http.MethodGet, "/api/v1/products/"+product.Id.String(), nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *ProductHandlerTestSuite) TestDeleteProductNotFound() {
	// Try to delete a non-existent product
	fakeUUID := "123e4567-e89b-12d3-a456-426614174000"
	rec, err := suite.makeRequest(http.MethodDelete, "/api/v1/products/"+fakeUUID, nil)
	suite.Require().NoError(err)

	suite.Equal(http.StatusNotFound, rec.Code)

	var errorResponse api.Error
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	suite.Require().NoError(err)

	suite.Equal("product_not_found", errorResponse.Error)
}

func (suite *ProductHandlerTestSuite) TestCreateProductValidation() {
	// Test with missing required fields
	createReq := api.CreateProductRequest{
		// Missing Title and Handle
	}

	rec, err := suite.makeRequest(http.MethodPost, "/api/v1/products", createReq)
	suite.Require().NoError(err)

	// Should return 400 Bad Request due to validation error
	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *ProductHandlerTestSuite) TestCreateProductWithAllFields() {
	flavorNotes := []string{"chocolate", "caramel", "nutty"}
	subscriptionIntervals := []api.SubscriptionInterval{api.Monthly, api.Biweekly}
	metadata := map[string]interface{}{
		"cupping_score": 88,
		"processing_notes": "Carefully processed",
	}

	createReq := api.CreateProductRequest{
		Title:                          "Premium Colombian",
		Handle:                         "premium-colombian",
		OriginCountry:                  stringPtr("Colombia"),
		Region:                         stringPtr("Huila"),
		RoastLevel:                     roastLevelPtr(api.Medium),
		ProcessingMethod:               processingMethodPtr(api.Washed),
		FlavorNotes:                    &flavorNotes,
		SubscriptionEnabled:            boolPtr(true),
		SubscriptionIntervals:          &subscriptionIntervals,
		SubscriptionDiscountPercentage: float64Ptr(10.0),
		Metadata:                       &metadata,
	}

	rec, err := suite.makeRequest(http.MethodPost, "/api/v1/products", createReq)
	suite.Require().NoError(err)

	suite.Equal(http.StatusCreated, rec.Code)

	var product api.Product
	err = json.Unmarshal(rec.Body.Bytes(), &product)
	suite.Require().NoError(err)

	suite.Equal("Premium Colombian", product.Title)
	suite.Equal("premium-colombian", product.Handle)
	suite.NotNil(product.OriginCountry)
	suite.Equal("Colombia", *product.OriginCountry)
	suite.NotNil(product.FlavorNotes)
	suite.Len(*product.FlavorNotes, 3)
	suite.Equal(true, product.SubscriptionEnabled)
}

// Helper functions for pointer creation
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool { return &b }
func float64Ptr(f float64) *float64 { return &f }
func roastLevelPtr(r api.RoastLevel) *api.RoastLevel { return &r }
func processingMethodPtr(p api.ProcessingMethod) *api.ProcessingMethod { return &p }

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set test environment
	_, err := config.Load("../../.env")
	if err != nil {
		log.Fatalf("Missing config, %v", err)
	}
	// Modify config for test database
	// cfg.DB.Name = suite.cfg.DB.Name + "_test"
	// os.Setenv("APP_ENV", "test")
	// os.Setenv("DB_NAME", "coffee_subscriptions") // Will become coffee_subscriptions_test
	
	// Run tests
	code := m.Run()
	
	// Exit with the same code as the tests
	os.Exit(code)
}

// TestProductHandlerSuite runs the test suite
func TestProductHandlerSuite(t *testing.T) {
	// Skip if no database connection is available
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(ProductHandlerTestSuite))
}