package service

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderService provides business logic for order operations
type OrderService interface {
	// CreateOrderFromPaymentIntent creates an order from a successful payment
	// This is the primary order creation flow for retail purchases
	// Implements idempotency via payment_intent_id to prevent duplicate orders
	CreateOrderFromPaymentIntent(ctx context.Context, paymentIntentID string) (*OrderDetail, error)

	// GetOrder retrieves a single order by ID with tenant scoping
	GetOrder(ctx context.Context, orderID string) (*OrderDetail, error)

	// GetOrderByNumber retrieves a single order by order number with tenant scoping
	GetOrderByNumber(ctx context.Context, orderNumber string) (*OrderDetail, error)
}

// OrderDetail aggregates order information with items and addresses
type OrderDetail struct {
	Order           repository.Order
	Items           []repository.OrderItem
	ShippingAddress repository.Address
	BillingAddress  repository.Address
	Payment         repository.Payment
}

type orderService struct {
	repo            repository.Querier
	tenantID        pgtype.UUID
	billingProvider billing.Provider
	shippingProvider shipping.Provider
}

// NewOrderService creates a new OrderService instance
// Requires billing and shipping providers for order creation flow
func NewOrderService(repo repository.Querier, tenantID string, billingProvider billing.Provider, shippingProvider shipping.Provider) (OrderService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &orderService{
		repo:             repo,
		tenantID:         tenantUUID,
		billingProvider:  billingProvider,
		shippingProvider: shippingProvider,
	}, nil
}

// CreateOrderFromPaymentIntent creates an order from a successful Stripe payment intent
// This method implements the complete order creation workflow with proper error handling
// and idempotency guarantees.
//
// Flow:
// 1. Idempotency check - prevent duplicate orders from webhook retries
// 2. Retrieve payment intent from Stripe
// 3. Validate payment status (must be succeeded)
// 4. Extract cart_id from payment metadata
// 5. Retrieve cart and validate tenant isolation
// 6. Load cart items with product details
// 7. Extract shipping/billing addresses from payment metadata
// 8. Begin database transaction for atomicity
// 9. Create address records (shipping and billing)
// 10. Create billing customer record (Stripe customer linkage)
// 11. Create payment method record (for saved cards)
// 12. Create payment record (transaction log)
// 13. Generate order number
// 14. Create order record
// 15. Create order items (snapshot cart state)
// 16. Decrement inventory for each SKU (with optimistic locking)
// 17. Mark cart as converted
// 18. Link payment to order
// 19. Commit transaction
// 20. Return complete order detail
//
// Idempotency:
// - Uses payment_intent_id as idempotency key
// - If order already exists for payment intent, returns existing order
// - Safe to call multiple times (webhooks may retry)
//
// Error Handling:
// - Returns ErrPaymentAlreadyProcessed if order exists for payment intent
// - Returns ErrPaymentNotSucceeded if payment status is not "succeeded"
// - Returns ErrMissingCartID if cart_id not found in payment metadata
// - Returns ErrCartNotFound if cart doesn't exist
// - Returns ErrTenantMismatch if cart tenant doesn't match service tenant
// - Returns ErrCartAlreadyConverted if cart already converted to order
// - Returns ErrInsufficientStock if any SKU lacks inventory
// - All database errors wrapped with context
func (s *orderService) CreateOrderFromPaymentIntent(ctx context.Context, paymentIntentID string) (*OrderDetail, error) {
	// TODO: Step 1 - Idempotency check
	// Call: s.repo.GetOrderByPaymentIntentID(ctx, repository.GetOrderByPaymentIntentIDParams{
	//         TenantID:          s.tenantID,
	//         ProviderPaymentID: paymentIntentID,
	//       })
	// If order found: Return existing order (convert to OrderDetail format)
	// If sql.ErrNoRows: Continue to step 2
	// If other error: Return wrapped error

	// TODO: Step 2 - Retrieve payment intent from Stripe
	// Call: s.billingProvider.GetPaymentIntent(ctx, billing.GetPaymentIntentParams{
	//         PaymentIntentID: paymentIntentID,
	//         TenantID:        s.tenantID.Bytes.String(), // Convert UUID to string
	//       })
	// Store result in variable: paymentIntent

	// TODO: Step 3 - Validate payment status
	// Check: paymentIntent.Status == "succeeded"
	// If not: Return ErrPaymentNotSucceeded

	// TODO: Step 4 - Extract cart_id from payment metadata
	// Get: cartID := paymentIntent.Metadata["cart_id"]
	// If empty: Return ErrMissingCartID
	// Parse into pgtype.UUID

	// TODO: Step 5 - Retrieve cart and validate tenant isolation
	// Call: s.repo.GetCartByID(ctx, cartUUID)
	// If sql.ErrNoRows: Return ErrCartNotFound
	// Check: cart.TenantID == s.tenantID (byte comparison)
	// If mismatch: Return ErrTenantMismatch
	// Check: cart.Status != "converted"
	// If already converted: Return ErrCartAlreadyConverted

	// TODO: Step 6 - Load cart items with product details
	// Call: s.repo.GetCartItems(ctx, cartUUID)
	// If no items: Return error "cart is empty"
	// Store in variable: cartItems

	// TODO: Step 7 - Extract and parse shipping/billing addresses from payment metadata
	// Stripe stores addresses in metadata as JSON strings
	// Get: shippingJSON := paymentIntent.Metadata["shipping_address"]
	// Get: billingJSON := paymentIntent.Metadata["billing_address"]
	// Parse each into temporary address structs
	// If parsing fails: Return wrapped error

	// TODO: Step 8 - Begin database transaction
	// Call: tx, err := s.repo.BeginTx(ctx)
	// Defer: tx.Rollback() (safe to call after commit)
	// Use tx.Queries() for all subsequent database operations

	// TODO: Step 9 - Create address records
	// Create shipping address:
	//   Call: txQueries.CreateAddress(ctx, repository.CreateAddressParams{
	//           TenantID:     s.tenantID,
	//           AddressType:  "shipping",
	//           FullName:     ...,
	//           AddressLine1: ...,
	//           ... (populate from parsed shipping address)
	//         })
	// Create billing address:
	//   Call: txQueries.CreateAddress(ctx, repository.CreateAddressParams{
	//           TenantID:     s.tenantID,
	//           AddressType:  "billing",
	//           ... (populate from parsed billing address)
	//         })
	// Store IDs: shippingAddressID, billingAddressID

	// TODO: Step 10 - Create billing customer record
	// Extract Stripe customer ID from payment intent
	// Get: stripeCustomerID := paymentIntent.Metadata["customer_id"] (or from payment intent CustomerID field)
	// Call: txQueries.CreateBillingCustomer(ctx, repository.CreateBillingCustomerParams{
	//         TenantID:           s.tenantID,
	//         UserID:            cart.UserID, // May be NULL for guest checkouts
	//         Provider:          "stripe",
	//         ProviderCustomerID: stripeCustomerID,
	//       })
	// Store: billingCustomer

	// TODO: Step 11 - Create payment method record (if saved)
	// This is for future subscriptions/saved cards
	// If payment intent has payment_method attached:
	//   Retrieve payment method details from Stripe
	//   Call: txQueries.CreatePaymentMethod(ctx, ...)
	// Store: paymentMethodID (may be NULL)

	// TODO: Step 12 - Create payment record
	// Call: txQueries.CreatePayment(ctx, repository.CreatePaymentParams{
	//         TenantID:          s.tenantID,
	//         BillingCustomerID: billingCustomer.ID,
	//         Provider:          "stripe",
	//         ProviderPaymentID: paymentIntentID,
	//         AmountCents:       paymentIntent.AmountCents,
	//         Currency:          paymentIntent.Currency,
	//         Status:            "succeeded",
	//         PaymentMethodID:   paymentMethodID, // From step 11
	//       })
	// Store: payment

	// TODO: Step 13 - Generate order number
	// Format: ORD-{timestamp}-{random}
	// Example: ORD-20250129-A3K9
	// Use: time.Now().Format("20060102") for date
	// Use: crypto/rand for random suffix (4 chars, alphanumeric)
	// Store: orderNumber

	// TODO: Step 14 - Create order record
	// Calculate totals from cart items:
	//   subtotalCents = sum(item.Quantity * item.UnitPriceCents)
	//   taxCents = paymentIntent.TaxCents (from Stripe Tax or manual calculation)
	//   shippingCents = paymentIntent.ShippingCents (from metadata)
	//   totalCents = subtotalCents + taxCents + shippingCents
	// Call: txQueries.CreateOrder(ctx, repository.CreateOrderParams{
	//         TenantID:          s.tenantID,
	//         CartID:            cart.ID,
	//         UserID:            cart.UserID, // May be NULL for guests
	//         OrderNumber:       orderNumber,
	//         OrderType:         "retail",
	//         Status:            "pending",
	//         SubtotalCents:     subtotalCents,
	//         ShippingCents:     shippingCents,
	//         TaxCents:          taxCents,
	//         TotalCents:        totalCents,
	//         Currency:          "usd",
	//         ShippingAddressID: shippingAddressID,
	//         BillingAddressID:  billingAddressID,
	//         CustomerNotes:     ... (from payment metadata if present),
	//       })
	// Store: order

	// TODO: Step 15 - Create order items
	// For each cart item:
	//   Build variant description from weight + grind
	//   Call: txQueries.CreateOrderItem(ctx, repository.CreateOrderItemParams{
	//           TenantID:           s.tenantID,
	//           OrderID:            order.ID,
	//           ProductSkuID:       item.ProductSkuID,
	//           ProductName:        item.ProductName,
	//           Sku:                item.Sku,
	//           VariantDescription: variantDesc, // "12oz - Whole Bean"
	//           Quantity:           item.Quantity,
	//           UnitPriceCents:     item.UnitPriceCents,
	//           TotalPriceCents:    item.Quantity * item.UnitPriceCents,
	//         })
	// Store in slice: orderItems

	// TODO: Step 16 - Decrement inventory for each SKU
	// For each cart item:
	//   Call: txQueries.DecrementSKUStock(ctx, repository.DecrementSKUStockParams{
	//           TenantID:          s.tenantID,
	//           ID:                item.ProductSkuID,
	//           InventoryQuantity: item.Quantity,
	//         })
	//   Check rows affected: If 0, return ErrInsufficientStock
	//   (The UPDATE includes WHERE inventory_quantity >= $3 for optimistic locking)

	// TODO: Step 17 - Mark cart as converted
	// Call: txQueries.UpdateCartStatus(ctx, repository.UpdateCartStatusParams{
	//         TenantID: s.tenantID,
	//         ID:       cart.ID,
	//         Status:   "converted",
	//       })

	// TODO: Step 18 - Link payment to order
	// Update order record with payment_id:
	//   Call: txQueries.UpdateOrderPaymentID(ctx, repository.UpdateOrderPaymentIDParams{
	//           TenantID:  s.tenantID,
	//           ID:        order.ID,
	//           PaymentID: payment.ID,
	//         })

	// TODO: Step 19 - Commit transaction
	// Call: tx.Commit(ctx)
	// If commit fails: Return wrapped error
	// Transaction is automatically rolled back if any step fails

	// TODO: Step 20 - Return complete order detail
	// Construct OrderDetail{
	//   Order:           order,
	//   Items:           orderItems,
	//   ShippingAddress: shippingAddress,
	//   BillingAddress:  billingAddress,
	//   Payment:         payment,
	// }

	return nil, fmt.Errorf("not implemented")
}

// GetOrder retrieves a single order by ID with all related data
func (s *orderService) GetOrder(ctx context.Context, orderID string) (*OrderDetail, error) {
	// TODO: Parse orderID into pgtype.UUID
	// TODO: Call s.repo.GetOrder(ctx, repository.GetOrderParams{
	//         TenantID: s.tenantID,
	//         ID:       orderUUID,
	//       })
	// TODO: If sql.ErrNoRows, return ErrOrderNotFound
	// TODO: Load related data:
	//       - Order items: s.repo.GetOrderItems(ctx, orderUUID)
	//       - Shipping address: s.repo.GetAddressByID(ctx, order.ShippingAddressID)
	//       - Billing address: s.repo.GetAddressByID(ctx, order.BillingAddressID)
	//       - Payment: s.repo.GetPaymentByID(ctx, order.PaymentID)
	// TODO: Construct and return OrderDetail

	return nil, fmt.Errorf("not implemented")
}

// GetOrderByNumber retrieves a single order by order number with all related data
func (s *orderService) GetOrderByNumber(ctx context.Context, orderNumber string) (*OrderDetail, error) {
	// TODO: Call s.repo.GetOrderByNumber(ctx, repository.GetOrderByNumberParams{
	//         TenantID:    s.tenantID,
	//         OrderNumber: orderNumber,
	//       })
	// TODO: If sql.ErrNoRows, return ErrOrderNotFound
	// TODO: Load related data (same as GetOrder):
	//       - Order items
	//       - Shipping address
	//       - Billing address
	//       - Payment
	// TODO: Construct and return OrderDetail

	return nil, fmt.Errorf("not implemented")
}
