package storefront

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionListHandler shows all subscriptions for the authenticated user
type SubscriptionListHandler struct {
	subscriptionService service.SubscriptionService
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
}

// NewSubscriptionListHandler creates a new subscription list handler
func NewSubscriptionListHandler(subscriptionService service.SubscriptionService, renderer *handler.Renderer, tenantID string) *SubscriptionListHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionListHandler{
		subscriptionService: subscriptionService,
		renderer:            renderer,
		tenantID:            tenantUUID,
	}
}

// ServeHTTP handles GET /account/subscriptions
func (h *SubscriptionListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context (RequireAuth middleware ensures this exists)
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/subscriptions", http.StatusSeeOther)
		return
	}

	subscriptions, err := h.subscriptionService.ListSubscriptionsForUser(ctx, service.ListSubscriptionsParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
		Limit:    50,
		Offset:   0,
	})

	if err != nil {
		http.Error(w, "Failed to load subscriptions", http.StatusInternalServerError)
		return
	}

	data := BaseTemplateData(r)
	data["Subscriptions"] = subscriptions

	h.renderer.RenderHTTP(w, "storefront/subscriptions", data)
}

// SubscriptionDetailHandler shows a single subscription for the authenticated user
type SubscriptionDetailHandler struct {
	subscriptionService service.SubscriptionService
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
}

// NewSubscriptionDetailHandler creates a new subscription detail handler
func NewSubscriptionDetailHandler(subscriptionService service.SubscriptionService, renderer *handler.Renderer, tenantID string) *SubscriptionDetailHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionDetailHandler{
		subscriptionService: subscriptionService,
		renderer:            renderer,
		tenantID:            tenantUUID,
	}
}

// ServeHTTP handles GET /account/subscriptions/{id}
func (h *SubscriptionDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get subscription ID from path
	subscriptionIDStr := r.PathValue("id")
	if subscriptionIDStr == "" {
		http.Error(w, "Subscription ID required", http.StatusBadRequest)
		return
	}

	var subscriptionID pgtype.UUID
	if err := subscriptionID.Scan(subscriptionIDStr); err != nil {
		http.Error(w, "Invalid subscription ID", http.StatusBadRequest)
		return
	}

	subscription, err := h.subscriptionService.GetSubscription(ctx, service.GetSubscriptionParams{
		TenantID:               h.tenantID,
		SubscriptionID:         subscriptionID,
		UserID:                 user.ID, // Include user ID for ownership validation
		IncludeUpcomingInvoice: true,
	})

	if err != nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}

	data := BaseTemplateData(r)
	data["Subscription"] = subscription

	h.renderer.RenderHTTP(w, "storefront/subscription_detail", data)
}

// SubscriptionPortalHandler creates a Stripe Customer Portal session and redirects
type SubscriptionPortalHandler struct {
	subscriptionService service.SubscriptionService
	tenantID            pgtype.UUID
}

// NewSubscriptionPortalHandler creates a new subscription portal handler
func NewSubscriptionPortalHandler(subscriptionService service.SubscriptionService, tenantID string) *SubscriptionPortalHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionPortalHandler{
		subscriptionService: subscriptionService,
		tenantID:            tenantUUID,
	}
}

// ServeHTTP handles GET /account/subscriptions/portal
func (h *SubscriptionPortalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/subscriptions/portal", http.StatusSeeOther)
		return
	}

	// Determine return URL
	returnURL := r.URL.Query().Get("return_to")
	if returnURL == "" {
		returnURL = "/account/subscriptions"
	}

	portalURL, err := h.subscriptionService.CreateCustomerPortalSession(ctx, service.PortalSessionParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		ReturnURL: returnURL,
	})

	if err != nil {
		http.Error(w, "Failed to create portal session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusSeeOther)
}

// SubscriptionCheckoutHandler displays the subscription checkout page
type SubscriptionCheckoutHandler struct {
	productService  service.ProductService
	accountService  service.AccountService
	renderer        *handler.Renderer
	tenantID        pgtype.UUID
}

// NewSubscriptionCheckoutHandler creates a new subscription checkout handler
func NewSubscriptionCheckoutHandler(
	productService service.ProductService,
	accountService service.AccountService,
	renderer *handler.Renderer,
	tenantID string,
) *SubscriptionCheckoutHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionCheckoutHandler{
		productService: productService,
		accountService: accountService,
		renderer:       renderer,
		tenantID:       tenantUUID,
	}
}

// ServeHTTP handles GET /subscribe/checkout
func (h *SubscriptionCheckoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		// Preserve the subscription parameters in return URL
		returnURL := fmt.Sprintf("/subscribe/checkout?%s", r.URL.RawQuery)
		http.Redirect(w, r, "/login?return_to="+returnURL, http.StatusSeeOther)
		return
	}

	// Get subscription parameters from query string
	skuID := r.URL.Query().Get("sku_id")
	quantityStr := r.URL.Query().Get("quantity")
	billingInterval := r.URL.Query().Get("billing_interval")

	if skuID == "" {
		http.Error(w, "Product SKU is required", http.StatusBadRequest)
		return
	}

	// Default values
	quantity := int32(1)
	if quantityStr != "" {
		q, err := strconv.Atoi(quantityStr)
		if err == nil && q > 0 {
			quantity = int32(q)
		}
	}

	if billingInterval == "" {
		billingInterval = service.BillingIntervalMonthly
	}

	// Validate billing interval
	if !service.IsValidBillingInterval(billingInterval) {
		http.Error(w, "Invalid billing interval", http.StatusBadRequest)
		return
	}

	// Get SKU details with product info
	skuDetail, err := h.productService.GetSKUForCheckout(ctx, skuID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Get user's saved addresses
	addresses, err := h.accountService.ListAddresses(ctx, h.tenantID, user.ID)
	if err != nil {
		// Continue with empty addresses - user can add new one
		addresses = []service.UserAddress{}
	}

	// Get user's saved payment methods
	paymentMethods, err := h.accountService.ListPaymentMethods(ctx, h.tenantID, user.ID)
	if err != nil {
		// Continue with empty payment methods - user can add new one
		paymentMethods = []service.UserPaymentMethod{}
	}

	// Calculate totals
	subtotalCents := skuDetail.PriceCents * quantity

	data := BaseTemplateData(r)
	data["SKU"] = skuDetail
	data["Quantity"] = quantity
	data["BillingInterval"] = billingInterval
	data["BillingIntervals"] = service.ValidBillingIntervals
	data["Addresses"] = addresses
	data["PaymentMethods"] = paymentMethods
	data["SubtotalCents"] = subtotalCents
	data["HasAddresses"] = len(addresses) > 0
	data["HasPaymentMethods"] = len(paymentMethods) > 0

	h.renderer.RenderHTTP(w, "storefront/subscription_checkout", data)
}

// CreateSubscriptionHandler handles subscription creation from checkout
type CreateSubscriptionHandler struct {
	subscriptionService service.SubscriptionService
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
}

// NewCreateSubscriptionHandler creates a new create subscription handler
func NewCreateSubscriptionHandler(subscriptionService service.SubscriptionService, renderer *handler.Renderer, tenantID string) *CreateSubscriptionHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &CreateSubscriptionHandler{
		subscriptionService: subscriptionService,
		renderer:            renderer,
		tenantID:            tenantUUID,
	}
}

// ServeHTTP handles POST /subscribe
func (h *CreateSubscriptionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Extract form fields
	productSKUIDStr := r.FormValue("product_sku_id")
	quantityStr := r.FormValue("quantity")
	billingInterval := r.FormValue("billing_interval")
	shippingAddressIDStr := r.FormValue("shipping_address_id")
	paymentMethodIDStr := r.FormValue("payment_method_id")

	// Validate required fields
	if productSKUIDStr == "" || billingInterval == "" || shippingAddressIDStr == "" || paymentMethodIDStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Parse quantity
	quantity := int32(1)
	if quantityStr != "" {
		q, err := strconv.Atoi(quantityStr)
		if err != nil || q < 1 {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}
		quantity = int32(q)
	}

	// Parse UUIDs
	var productSKUID, shippingAddressID, paymentMethodID pgtype.UUID
	if err := productSKUID.Scan(productSKUIDStr); err != nil {
		http.Error(w, "Invalid product SKU ID", http.StatusBadRequest)
		return
	}
	if err := shippingAddressID.Scan(shippingAddressIDStr); err != nil {
		http.Error(w, "Invalid shipping address ID", http.StatusBadRequest)
		return
	}
	if err := paymentMethodID.Scan(paymentMethodIDStr); err != nil {
		http.Error(w, "Invalid payment method ID", http.StatusBadRequest)
		return
	}

	// Validate billing interval
	if !service.IsValidBillingInterval(billingInterval) {
		http.Error(w, "Invalid billing interval", http.StatusBadRequest)
		return
	}

	// Create subscription
	subscription, err := h.subscriptionService.CreateSubscription(ctx, service.CreateSubscriptionParams{
		TenantID:          h.tenantID,
		UserID:            user.ID,
		ProductSKUID:      productSKUID,
		Quantity:          quantity,
		BillingInterval:   billingInterval,
		ShippingAddressID: shippingAddressID,
		PaymentMethodID:   paymentMethodID,
	})

	if err != nil {
		// TODO: Better error handling with user-friendly messages
		http.Error(w, fmt.Sprintf("Failed to create subscription: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to subscription detail page
	subscriptionIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		subscription.ID.Bytes[0:4], subscription.ID.Bytes[4:6], subscription.ID.Bytes[6:8],
		subscription.ID.Bytes[8:10], subscription.ID.Bytes[10:16])
	http.Redirect(w, r, "/account/subscriptions/"+subscriptionIDStr, http.StatusSeeOther)
}
