package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/crypto"
	"github.com/dukerupert/freyja/internal/email"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/handler/storefront"
	"github.com/dukerupert/freyja/internal/provider"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// IntegrationsHandler handles provider integration configuration routes
type IntegrationsHandler struct {
	repo      repository.Querier
	renderer  *handler.Renderer
	tenantID  pgtype.UUID
	encryptor crypto.Encryptor
	validator *provider.DefaultValidator
	registry  provider.ProviderRegistry
}

// NewIntegrationsHandler creates a new integrations handler
func NewIntegrationsHandler(
	repo repository.Querier,
	renderer *handler.Renderer,
	tenantID string,
	encryptor crypto.Encryptor,
	validator *provider.DefaultValidator,
	registry provider.ProviderRegistry,
) *IntegrationsHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &IntegrationsHandler{
		repo:      repo,
		renderer:  renderer,
		tenantID:  tenantUUID,
		encryptor: encryptor,
		validator: validator,
		registry:  registry,
	}
}

// ProviderSummary represents a provider configuration summary for display
type ProviderSummary struct {
	Type         string
	ProviderName string
	IsConfigured bool
	IsActive     bool
}

// ListPage handles GET /admin/settings/integrations
func (h *IntegrationsHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	providerTypes := []provider.ProviderType{
		provider.ProviderTypeBilling,
		provider.ProviderTypeTax,
		provider.ProviderTypeShipping,
		provider.ProviderTypeEmail,
	}

	summaries := make([]ProviderSummary, 0, len(providerTypes))

	for _, providerType := range providerTypes {
		config, err := h.repo.GetDefaultProviderConfig(ctx, repository.GetDefaultProviderConfigParams{
			TenantID: h.tenantID,
			Type:     string(providerType),
		})

		summary := ProviderSummary{
			Type: string(providerType),
		}

		if err == nil && config.ID.Valid {
			summary.ProviderName = config.Name
			summary.IsConfigured = true
			summary.IsActive = config.IsActive
		} else {
			summary.ProviderName = "none"
			summary.IsConfigured = false
			summary.IsActive = false
		}

		summaries = append(summaries, summary)
	}

	data := storefront.BaseTemplateData(r)
	data["CurrentPath"] = r.URL.Path
	data["Providers"] = summaries

	h.renderer.RenderHTTP(w, "admin/integrations", data)
}

// ConfigPage handles GET /admin/settings/integrations/{type}
func (h *IntegrationsHandler) ConfigPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	providerTypeStr := r.PathValue("type")
	if providerTypeStr == "" {
		http.Error(w, "Provider type required", http.StatusBadRequest)
		return
	}

	providerType := provider.ProviderType(providerTypeStr)

	config, err := h.repo.GetDefaultProviderConfig(ctx, repository.GetDefaultProviderConfigParams{
		TenantID: h.tenantID,
		Type:     string(providerType),
	})

	var currentProvider string
	var configMap map[string]interface{}
	var configID pgtype.UUID

	var configCorrupted bool
	if err == nil && config.ID.Valid {
		currentProvider = config.Name
		configID = config.ID

		if config.ConfigEncrypted != "" {
			decrypted, decryptErr := h.encryptor.Decrypt([]byte(config.ConfigEncrypted))
			if decryptErr != nil {
				// Log error without exposing sensitive data
				slog.Error("failed to decrypt provider configuration",
					slog.String("provider_type", string(providerType)),
					slog.String("error", decryptErr.Error()))
				configCorrupted = true
			} else {
				if unmarshalErr := json.Unmarshal(decrypted, &configMap); unmarshalErr != nil {
					slog.Error("failed to unmarshal provider configuration",
						slog.String("provider_type", string(providerType)),
						slog.String("error", unmarshalErr.Error()))
					configCorrupted = true
				}
			}
		}
	}

	if configMap == nil {
		configMap = make(map[string]interface{})
	}

	maskedConfig := maskSecrets(configMap)
	providerOptions := getProviderOptions(providerType)

	// Default to first provider option if none configured
	if currentProvider == "" && len(providerOptions) > 0 {
		currentProvider = providerOptions[0]["Value"]
	}

	data := storefront.BaseTemplateData(r)
	data["CurrentPath"] = r.URL.Path
	data["ProviderType"] = string(providerType)
	data["CurrentProvider"] = currentProvider
	data["ConfigID"] = configID
	data["Config"] = maskedConfig
	data["ProviderOptions"] = providerOptions
	data["ConfigCorrupted"] = configCorrupted

	h.renderer.RenderHTTP(w, "admin/integration_config", data)
}

// SaveConfig handles POST /admin/settings/integrations/{type}
func (h *IntegrationsHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	providerTypeStr := r.PathValue("type")
	if providerTypeStr == "" {
		http.Error(w, "Provider type required", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	providerType := provider.ProviderType(providerTypeStr)
	providerName := provider.ProviderName(strings.TrimSpace(r.FormValue("provider_name")))

	// Validate that provider name is appropriate for the provider type
	// This prevents mismatched configurations (e.g., setting stripe_tax as billing provider)
	if !provider.IsValidProviderNameForType(providerName, providerType) {
		http.Error(w, fmt.Sprintf("Invalid provider %q for type %q", providerName, providerType), http.StatusBadRequest)
		return
	}

	configMap := buildConfigMap(r, providerType, providerName)

	// Validate that required credentials are present (not empty)
	// This prevents credential rotation bypass via empty form submissions
	if missingCreds := getMissingCredentials(providerName, configMap); len(missingCreds) > 0 {
		http.Error(w, fmt.Sprintf("Missing required credentials: %s", strings.Join(missingCreds, ", ")), http.StatusBadRequest)
		return
	}

	tenantConfig := &provider.TenantProviderConfig{
		TenantID:  h.tenantID,
		Type:      providerType,
		Name:      providerName,
		Config:    configMap,
		IsActive:  true,
		IsDefault: true,
		Priority:  0,
	}

	validationResult := h.validateConfig(tenantConfig)
	if !validationResult.Valid {
		errorMsg := strings.Join(validationResult.Errors, ", ")
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	configJSON, err := json.Marshal(configMap)
	if err != nil {
		http.Error(w, "Failed to encode configuration", http.StatusInternalServerError)
		return
	}

	encryptedConfig, err := h.encryptor.Encrypt(configJSON)
	if err != nil {
		http.Error(w, "Failed to encrypt configuration", http.StatusInternalServerError)
		return
	}

	// Invalidate cache BEFORE database update to ensure subsequent requests
	// will fetch fresh config. This prevents stale data between update and invalidation.
	h.registry.InvalidateCache(h.tenantID, providerType)

	existingConfig, err := h.repo.GetDefaultProviderConfig(ctx, repository.GetDefaultProviderConfigParams{
		TenantID: h.tenantID,
		Type:     string(providerType),
	})

	if err == nil && existingConfig.ID.Valid {
		// Verify tenant ownership before updating
		if existingConfig.TenantID != h.tenantID {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		_, err = h.repo.UpdateProviderConfig(ctx, repository.UpdateProviderConfigParams{
			ID:       existingConfig.ID,
			TenantID: h.tenantID,
			Name: pgtype.Text{
				String: string(providerName),
				Valid:  true,
			},
			IsActive: pgtype.Bool{
				Bool:  true,
				Valid: true,
			},
			IsDefault: pgtype.Bool{
				Bool:  true,
				Valid: true,
			},
			Priority: pgtype.Int4{
				Int32: 0,
				Valid: true,
			},
			ConfigEncrypted: pgtype.Text{
				String: string(encryptedConfig),
				Valid:  true,
			},
		})
		if err != nil {
			http.Error(w, "Failed to update configuration", http.StatusInternalServerError)
			return
		}
	} else {
		_, err = h.repo.CreateProviderConfig(ctx, repository.CreateProviderConfigParams{
			TenantID:        h.tenantID,
			Type:            string(providerType),
			Name:            string(providerName),
			IsActive:        true,
			IsDefault:       true,
			Priority:        0,
			ConfigEncrypted: string(encryptedConfig),
		})
		if err != nil {
			http.Error(w, "Failed to create configuration", http.StatusInternalServerError)
			return
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/admin/settings/integrations")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/admin/settings/integrations", http.StatusSeeOther)
}

// ValidateConfig handles POST /admin/settings/integrations/{type}/validate
func (h *IntegrationsHandler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	providerTypeStr := r.PathValue("type")
	if providerTypeStr == "" {
		http.Error(w, "Provider type required", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	providerType := provider.ProviderType(providerTypeStr)
	providerName := provider.ProviderName(strings.TrimSpace(r.FormValue("provider_name")))

	configMap := buildConfigMap(r, providerType, providerName)

	tenantConfig := &provider.TenantProviderConfig{
		TenantID: h.tenantID,
		Type:     providerType,
		Name:     providerName,
		Config:   configMap,
	}

	validationResult := h.validateConfig(tenantConfig)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":  validationResult.Valid,
		"errors": validationResult.Errors,
	})
}

// validateConfig validates provider configuration based on type
func (h *IntegrationsHandler) validateConfig(config *provider.TenantProviderConfig) *provider.ValidationResult {
	switch config.Type {
	case provider.ProviderTypeTax:
		return h.validator.ValidateTaxConfig(config)
	case provider.ProviderTypeShipping:
		return h.validator.ValidateShippingConfig(config)
	case provider.ProviderTypeBilling:
		return h.validator.ValidateBillingConfig(config)
	case provider.ProviderTypeEmail:
		return h.validator.ValidateEmailConfig(config)
	default:
		result := &provider.ValidationResult{Valid: false}
		result.AddError("unknown provider type")
		return result
	}
}

// buildConfigMap builds configuration map from form values based on provider
func buildConfigMap(r *http.Request, providerType provider.ProviderType, providerName provider.ProviderName) map[string]interface{} {
	configMap := make(map[string]interface{})

	switch providerName {
	case provider.ProviderNameStripe:
		if publishableKey := strings.TrimSpace(r.FormValue("stripe_publishable_key")); publishableKey != "" {
			configMap["stripe_publishable_key"] = publishableKey
		}
		if apiKey := strings.TrimSpace(r.FormValue("stripe_api_key")); apiKey != "" {
			configMap["stripe_api_key"] = apiKey
		}
		if webhookSecret := strings.TrimSpace(r.FormValue("stripe_webhook_secret")); webhookSecret != "" {
			configMap["stripe_webhook_secret"] = webhookSecret
		}

	case provider.ProviderNameStripeTax:
		if apiKey := strings.TrimSpace(r.FormValue("stripe_api_key")); apiKey != "" {
			configMap["stripe_api_key"] = apiKey
		}

	case provider.ProviderNameEasyPost:
		if apiKey := strings.TrimSpace(r.FormValue("easypost_api_key")); apiKey != "" {
			configMap["easypost_api_key"] = apiKey
		}

	case provider.ProviderNameShipStation:
		if apiKey := strings.TrimSpace(r.FormValue("api_key")); apiKey != "" {
			configMap["api_key"] = apiKey
		}
		if apiSecret := strings.TrimSpace(r.FormValue("api_secret")); apiSecret != "" {
			configMap["api_secret"] = apiSecret
		}

	case provider.ProviderNameShippo:
		if apiKey := strings.TrimSpace(r.FormValue("api_key")); apiKey != "" {
			configMap["api_key"] = apiKey
		}

	case provider.ProviderNamePostmark:
		if apiKey := strings.TrimSpace(r.FormValue("postmark_api_key")); apiKey != "" {
			configMap["postmark_api_key"] = apiKey
		}

	case provider.ProviderNameResend:
		if apiKey := strings.TrimSpace(r.FormValue("api_key")); apiKey != "" {
			configMap["api_key"] = apiKey
		}

	case provider.ProviderNameSES:
		if accessKey := strings.TrimSpace(r.FormValue("access_key_id")); accessKey != "" {
			configMap["access_key_id"] = accessKey
		}
		if secretKey := strings.TrimSpace(r.FormValue("secret_access_key")); secretKey != "" {
			configMap["secret_access_key"] = secretKey
		}
		if region := strings.TrimSpace(r.FormValue("region")); region != "" {
			configMap["region"] = region
		}

	case provider.ProviderNameSMTP:
		if host := strings.TrimSpace(r.FormValue("smtp_host")); host != "" {
			configMap["smtp_host"] = host
		}
		if portStr := strings.TrimSpace(r.FormValue("smtp_port")); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				configMap["smtp_port"] = port
			}
		}
		if username := strings.TrimSpace(r.FormValue("smtp_username")); username != "" {
			configMap["smtp_username"] = username
		}
		if password := strings.TrimSpace(r.FormValue("smtp_password")); password != "" {
			configMap["smtp_password"] = password
		}
		if from := strings.TrimSpace(r.FormValue("smtp_from")); from != "" {
			configMap["smtp_from"] = from
		}

	case provider.ProviderNameTaxJar:
		if apiKey := strings.TrimSpace(r.FormValue("api_key")); apiKey != "" {
			configMap["api_key"] = apiKey
		}

	case provider.ProviderNameAvalara:
		if accountID := strings.TrimSpace(r.FormValue("account_id")); accountID != "" {
			configMap["account_id"] = accountID
		}
		if licenseKey := strings.TrimSpace(r.FormValue("license_key")); licenseKey != "" {
			configMap["license_key"] = licenseKey
		}
	}

	return configMap
}

// maskSecrets replaces secret values with masked placeholder
func maskSecrets(config map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{})

	secretKeys := map[string]bool{
		"stripe_api_key":        true,
		"stripe_webhook_secret": true,
		"easypost_api_key":      true,
		"api_key":               true,
		"api_secret":            true,
		"postmark_api_key":      true,
		"secret_access_key":     true,
		"smtp_password":         true,
		"license_key":           true,
	}

	for key, value := range config {
		if secretKeys[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				masked[key] = "••••••••"
			} else {
				masked[key] = ""
			}
		} else {
			masked[key] = value
		}
	}

	return masked
}

// getProviderOptions returns available provider options for a given type
func getProviderOptions(providerType provider.ProviderType) []map[string]string {
	switch providerType {
	case provider.ProviderTypeTax:
		return []map[string]string{
			{"Value": string(provider.ProviderNameNoTax), "Label": "None"},
			{"Value": string(provider.ProviderNamePercentage), "Label": "Percentage (Database)"},
			{"Value": string(provider.ProviderNameStripeTax), "Label": "Stripe Tax"},
			{"Value": string(provider.ProviderNameTaxJar), "Label": "TaxJar"},
			{"Value": string(provider.ProviderNameAvalara), "Label": "Avalara"},
		}

	case provider.ProviderTypeShipping:
		return []map[string]string{
			{"Value": string(provider.ProviderNameManual), "Label": "Flat Rate (Manual)"},
			{"Value": string(provider.ProviderNameEasyPost), "Label": "EasyPost"},
			{"Value": string(provider.ProviderNameShipStation), "Label": "ShipStation"},
			{"Value": string(provider.ProviderNameShippo), "Label": "Shippo"},
		}

	case provider.ProviderTypeBilling:
		return []map[string]string{
			{"Value": string(provider.ProviderNameStripe), "Label": "Stripe"},
		}

	case provider.ProviderTypeEmail:
		return []map[string]string{
			{"Value": string(provider.ProviderNameSMTP), "Label": "SMTP"},
			{"Value": string(provider.ProviderNamePostmark), "Label": "Postmark"},
			{"Value": string(provider.ProviderNameResend), "Label": "Resend"},
			{"Value": string(provider.ProviderNameSES), "Label": "Amazon SES"},
		}

	default:
		return []map[string]string{}
	}
}

// TestConnection handles POST /admin/settings/integrations/{type}/test
// It tests the connection to the provider using the provided credentials.
func (h *IntegrationsHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	providerTypeStr := r.PathValue("type")
	if providerTypeStr == "" {
		writeTestConnectionResponse(w, false, "Provider type required")
		return
	}

	// For multipart form data, we need to use ParseMultipartForm
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeTestConnectionResponse(w, false, "Invalid form data")
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			writeTestConnectionResponse(w, false, "Invalid form data")
			return
		}
	}

	providerType := provider.ProviderType(providerTypeStr)
	providerName := provider.ProviderName(strings.TrimSpace(r.FormValue("provider_name")))

	configMap := buildConfigMap(r, providerType, providerName)

	// Test the connection based on provider type
	var testErr error
	switch providerType {
	case provider.ProviderTypeBilling:
		testErr = h.testBillingConnection(providerName, configMap)
	case provider.ProviderTypeTax:
		testErr = h.testTaxConnection(providerName, configMap)
	case provider.ProviderTypeShipping:
		testErr = h.testShippingConnection(providerName, configMap)
	case provider.ProviderTypeEmail:
		testErr = h.testEmailConnection(providerName, configMap)
	default:
		writeTestConnectionResponse(w, false, "Unknown provider type")
		return
	}

	if testErr != nil {
		writeTestConnectionResponse(w, false, testErr.Error())
		return
	}

	writeTestConnectionResponse(w, true, "Connection successful")
}

// testBillingConnection tests a billing provider connection
func (h *IntegrationsHandler) testBillingConnection(name provider.ProviderName, config map[string]interface{}) error {
	switch name {
	case provider.ProviderNameStripe:
		apiKey, ok := config["stripe_api_key"].(string)
		if !ok || apiKey == "" {
			return fmt.Errorf("Stripe API key is required")
		}
		// Test by creating a temporary Stripe provider and making a simple API call
		return testStripeAPIKey(apiKey)
	default:
		return fmt.Errorf("unsupported billing provider: %s", name)
	}
}

// testTaxConnection tests a tax provider connection
func (h *IntegrationsHandler) testTaxConnection(name provider.ProviderName, config map[string]interface{}) error {
	switch name {
	case provider.ProviderNameNoTax, provider.ProviderNamePercentage:
		// No external connection needed
		return nil
	case provider.ProviderNameStripeTax:
		// Stripe Tax uses billing Stripe credentials - no separate test needed
		return nil
	case provider.ProviderNameTaxJar, provider.ProviderNameAvalara:
		return fmt.Errorf("test connection not implemented for %s", name)
	default:
		return fmt.Errorf("unsupported tax provider: %s", name)
	}
}

// testShippingConnection tests a shipping provider connection
func (h *IntegrationsHandler) testShippingConnection(name provider.ProviderName, config map[string]interface{}) error {
	switch name {
	case provider.ProviderNameManual:
		// No external connection needed
		return nil
	case provider.ProviderNameEasyPost:
		apiKey, ok := config["easypost_api_key"].(string)
		if !ok || apiKey == "" {
			return fmt.Errorf("EasyPost API key is required")
		}
		return testEasyPostAPIKey(apiKey)
	case provider.ProviderNameShipStation, provider.ProviderNameShippo:
		return fmt.Errorf("test connection not implemented for %s", name)
	default:
		return fmt.Errorf("unsupported shipping provider: %s", name)
	}
}

// testEmailConnection tests an email provider connection
func (h *IntegrationsHandler) testEmailConnection(name provider.ProviderName, config map[string]interface{}) error {
	switch name {
	case provider.ProviderNamePostmark:
		apiKey, ok := config["postmark_api_key"].(string)
		if !ok || apiKey == "" {
			return fmt.Errorf("Postmark API key is required")
		}
		return testPostmarkAPIKey(apiKey)
	case provider.ProviderNameSMTP:
		host, ok := config["smtp_host"].(string)
		if !ok || host == "" {
			return fmt.Errorf("SMTP host is required")
		}
		portStr, ok := config["smtp_port"].(string)
		if !ok || portStr == "" {
			return fmt.Errorf("SMTP port is required")
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid SMTP port: %w", err)
		}
		username, _ := config["smtp_username"].(string)
		password, _ := config["smtp_password"].(string)
		return email.TestSMTPConnection(host, port, username, password)
	case provider.ProviderNameResend, provider.ProviderNameSES:
		return fmt.Errorf("test connection not implemented for %s", name)
	default:
		return fmt.Errorf("unsupported email provider: %s", name)
	}
}

// testStripeAPIKey tests a Stripe API key by making a simple API call
func testStripeAPIKey(apiKey string) error {
	// Use the Stripe Go SDK to test the API key
	// We'll make a simple balance retrieve call which requires minimal permissions
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.stripe.com/v1/balance", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(apiKey, "")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// testPostmarkAPIKey tests a Postmark API key by calling the Server API endpoint
func testPostmarkAPIKey(apiKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.postmarkapp.com/server", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Postmark-Server-Token", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API token")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// testEasyPostAPIKey tests an EasyPost API key by creating a test address
func testEasyPostAPIKey(apiKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a minimal address verification request
	// This works with both test and production API keys
	payload := `{"address":{"street1":"417 MONTGOMERY ST","city":"SAN FRANCISCO","state":"CA","zip":"94104","country":"US"}}`

	req, err := http.NewRequest("POST", "https://api.easypost.com/v2/addresses", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// writeTestConnectionResponse writes a JSON response for test connection requests
func writeTestConnectionResponse(w http.ResponseWriter, success bool, message string) {
	w.Header().Set("Content-Type", "application/json")
	if !success {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": message,
	})
}

// getMissingCredentials returns a list of required credential fields that are missing from configMap.
// This is used to reject empty form submissions that would bypass credential rotation.
func getMissingCredentials(providerName provider.ProviderName, configMap map[string]interface{}) []string {
	var requiredFields []string

	switch providerName {
	case provider.ProviderNameStripe:
		requiredFields = []string{"stripe_publishable_key", "stripe_api_key", "stripe_webhook_secret"}
	case provider.ProviderNameEasyPost:
		requiredFields = []string{"easypost_api_key"}
	case provider.ProviderNameShipStation:
		requiredFields = []string{"api_key", "api_secret"}
	case provider.ProviderNameShippo:
		requiredFields = []string{"api_key"}
	case provider.ProviderNamePostmark:
		requiredFields = []string{"postmark_api_key"}
	case provider.ProviderNameResend:
		requiredFields = []string{"api_key"}
	case provider.ProviderNameSES:
		requiredFields = []string{"access_key_id", "secret_access_key", "region"}
	case provider.ProviderNameSMTP:
		requiredFields = []string{"smtp_host", "smtp_port"}
	case provider.ProviderNameTaxJar:
		requiredFields = []string{"api_key"}
	case provider.ProviderNameAvalara:
		requiredFields = []string{"account_id", "license_key"}
	case provider.ProviderNameNoTax, provider.ProviderNamePercentage, provider.ProviderNameManual, provider.ProviderNameStripeTax:
		// These providers don't require credentials
		// stripe_tax uses the billing provider's Stripe API key
		return nil
	default:
		return nil
	}

	var missing []string
	for _, field := range requiredFields {
		if _, exists := configMap[field]; !exists {
			missing = append(missing, field)
		}
	}

	return missing
}
