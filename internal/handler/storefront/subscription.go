package storefront

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionHandler handles all account-related subscription operations:
// - Subscription listing
// - Subscription detail view
// - Customer portal redirect
// - Subscription checkout
// - Subscription creation
type SubscriptionHandler struct {
	subscriptionService service.SubscriptionService
	productService      service.ProductService
	accountService      service.AccountService
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
}

// NewSubscriptionHandler creates a new consolidated subscription handler
func NewSubscriptionHandler(
	subscriptionService service.SubscriptionService,
	productService service.ProductService,
	accountService service.AccountService,
	renderer *handler.Renderer,
	tenantID string,
) *SubscriptionHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
		productService:      productService,
		accountService:      accountService,
		renderer:            renderer,
		tenantID:            tenantUUID,
	}
}

// =============================================================================
// Subscription List
// =============================================================================

// List handles GET /account/subscriptions - shows all subscriptions for the user
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
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
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := BaseTemplateData(r)
	data["Subscriptions"] = subscriptions

	h.renderer.RenderHTTP(w, "storefront/subscriptions", data)
}

// =============================================================================
// Subscription Detail
// =============================================================================

// Detail handles GET /account/subscriptions/{id} - shows a single subscription
func (h *SubscriptionHandler) Detail(w http.ResponseWriter, r *http.Request) {
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Subscription ID required"))
		return
	}

	var subscriptionID pgtype.UUID
	if err := subscriptionID.Scan(subscriptionIDStr); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid subscription ID"))
		return
	}

	subscription, err := h.subscriptionService.GetSubscription(ctx, service.GetSubscriptionParams{
		TenantID:               h.tenantID,
		SubscriptionID:         subscriptionID,
		UserID:                 user.ID, // Include user ID for ownership validation
		IncludeUpcomingInvoice: true,
	})

	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	data := BaseTemplateData(r)
	data["Subscription"] = subscription

	h.renderer.RenderHTTP(w, "storefront/subscription_detail", data)
}

// =============================================================================
// Customer Portal
// =============================================================================

// Portal handles GET /account/subscriptions/portal - redirects to Stripe portal
func (h *SubscriptionHandler) Portal(w http.ResponseWriter, r *http.Request) {
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
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusSeeOther)
}

// =============================================================================
// Subscription Checkout
// =============================================================================

// Checkout handles GET /subscribe/checkout - shows subscription checkout page
func (h *SubscriptionHandler) Checkout(w http.ResponseWriter, r *http.Request) {
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Product SKU is required"))
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid billing interval"))
		return
	}

	// Get SKU details with product info
	skuDetail, err := h.productService.GetSKUForCheckout(ctx, skuID)
	if err != nil {
		handler.NotFoundResponse(w, r)
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

// =============================================================================
// Create Subscription
// =============================================================================

// Create handles POST /subscribe - creates a new subscription
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Missing required fields"))
		return
	}

	// Parse quantity
	quantity := int32(1)
	if quantityStr != "" {
		q, err := strconv.Atoi(quantityStr)
		if err != nil || q < 1 {
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid quantity"))
			return
		}
		quantity = int32(q)
	}

	// Parse UUIDs
	var productSKUID, shippingAddressID, paymentMethodID pgtype.UUID
	if err := productSKUID.Scan(productSKUIDStr); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product SKU ID"))
		return
	}
	if err := shippingAddressID.Scan(shippingAddressIDStr); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid shipping address ID"))
		return
	}
	if err := paymentMethodID.Scan(paymentMethodIDStr); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid payment method ID"))
		return
	}

	// Validate billing interval
	if !service.IsValidBillingInterval(billingInterval) {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid billing interval"))
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
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Redirect to subscription detail page
	subscriptionIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		subscription.ID.Bytes[0:4], subscription.ID.Bytes[4:6], subscription.ID.Bytes[6:8],
		subscription.ID.Bytes[8:10], subscription.ID.Bytes[10:16])
	http.Redirect(w, r, "/account/subscriptions/"+subscriptionIDStr, http.StatusSeeOther)
}
