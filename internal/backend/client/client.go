// internal/backend/client/client.go
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"encoding/json"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type FreyjaClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewFreyjaClient(baseURL string) *FreyjaClient {
	return &FreyjaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProducts fetches all products from the Freyja API
func (c *FreyjaClient) GetProducts(ctx context.Context) ([]interfaces.Product, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/products", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response struct {
		Products []interfaces.Product `json:"products"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal products: %w", err)
	}

	return response.Products, nil
}

// CreateProductFromForm forwards the form data directly to the server
func (c *FreyjaClient) CreateProductFromForm(ctx context.Context, originalReq *http.Request) error {
	// Parse the original form
	if err := originalReq.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	// Create a new form with the same data
	formData := url.Values{}
	for key, values := range originalReq.Form {
		for _, value := range values {
			formData.Add(key, value)
		}
	}

	// Create request to server's admin endpoint
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/api/v1/admin/products",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true") // Forward HTMX header if present
	if originalReq.Header.Get("HX-Request") != "" {
		req.Header.Set("HX-Request", originalReq.Header.Get("HX-Request"))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateProductFromForm forwards form data to update an existing product
func (c *FreyjaClient) UpdateProductFromForm(ctx context.Context, productID string, originalReq *http.Request) error {
	// Parse the original form
	if err := originalReq.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	// Create a new form with the same data
	formData := url.Values{}
	for key, values := range originalReq.Form {
		for _, value := range values {
			formData.Add(key, value)
		}
	}

	// Create request to server's admin endpoint
	req, err := http.NewRequestWithContext(
		ctx,
		"PUT",
		fmt.Sprintf("%s/api/v1/admin/products/%s", c.baseURL, productID),
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if originalReq.Header.Get("HX-Request") != "" {
		req.Header.Set("HX-Request", originalReq.Header.Get("HX-Request"))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteProduct deletes a product via the Freyja API
func (c *FreyjaClient) DeleteProduct(ctx context.Context, productID string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		fmt.Sprintf("%s/api/v1/admin/products/%s", c.baseURL, productID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}