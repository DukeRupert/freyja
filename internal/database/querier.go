// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type Querier interface {
	ActivateProduct(ctx context.Context, id int32) (Products, error)
	ArchiveCustomer(ctx context.Context, id int32) (ArchiveCustomerRow, error)
	ClearCartItems(ctx context.Context, cartID int32) error
	CreateCart(ctx context.Context, arg CreateCartParams) (Carts, error)
	CreateCartItem(ctx context.Context, arg CreateCartItemParams) (CartItems, error)
	// internal/database/queries/customers.sql
	CreateCustomer(ctx context.Context, arg CreateCustomerParams) (CreateCustomerRow, error)
	CreateOrder(ctx context.Context, arg CreateOrderParams) (CreateOrderRow, error)
	CreateOrderItem(ctx context.Context, arg CreateOrderItemParams) (CreateOrderItemRow, error)
	CreateProduct(ctx context.Context, arg CreateProductParams) (Products, error)
	DeactivateProduct(ctx context.Context, id int32) (Products, error)
	DecrementProductStock(ctx context.Context, arg DecrementProductStockParams) (Products, error)
	DeleteCart(ctx context.Context, id int32) error
	DeleteCartItem(ctx context.Context, id int32) error
	DeleteCartItemByProductAndType(ctx context.Context, arg DeleteCartItemByProductAndTypeParams) error
	DeleteCartItemByProductID(ctx context.Context, arg DeleteCartItemByProductIDParams) error
	DeleteCustomer(ctx context.Context, id int32) error
	DeleteProduct(ctx context.Context, id int32) error
	GetAllOrders(ctx context.Context, arg GetAllOrdersParams) ([]GetAllOrdersRow, error)
	GetAllOrdersWithFilters(ctx context.Context, arg GetAllOrdersWithFiltersParams) ([]Orders, error)
	GetArchivedCustomers(ctx context.Context, arg GetArchivedCustomersParams) ([]Customers, error)
	// internal/database/queries/carts.sql
	GetCart(ctx context.Context, id int32) (Carts, error)
	GetCartByCustomerID(ctx context.Context, customerID pgtype.Int4) (Carts, error)
	GetCartBySessionID(ctx context.Context, sessionID pgtype.Text) (Carts, error)
	// internal/database/queries/cart_items.sql
	GetCartItem(ctx context.Context, id int32) (CartItems, error)
	GetCartItemByProductAndType(ctx context.Context, arg GetCartItemByProductAndTypeParams) (CartItems, error)
	GetCartItemByProductID(ctx context.Context, arg GetCartItemByProductIDParams) (CartItems, error)
	GetCartItemCount(ctx context.Context, cartID int32) (int32, error)
	GetCartItems(ctx context.Context, cartID int32) ([]GetCartItemsRow, error)
	GetCartItemsByProduct(ctx context.Context, arg GetCartItemsByProductParams) ([]CartItems, error)
	GetCartItemsByPurchaseType(ctx context.Context, arg GetCartItemsByPurchaseTypeParams) ([]GetCartItemsByPurchaseTypeRow, error)
	GetCartTotal(ctx context.Context, cartID int32) (int32, error)
	GetCustomer(ctx context.Context, id int32) (Customers, error)
	GetCustomerByEmail(ctx context.Context, lower string) (Customers, error)
	GetCustomerByStripeID(ctx context.Context, stripeCustomerID pgtype.Text) (Customers, error)
	GetCustomerCount(ctx context.Context) (int64, error)
	GetCustomerCountWithStripeID(ctx context.Context) (int64, error)
	GetCustomerOrderStats(ctx context.Context, customerID int32) (GetCustomerOrderStatsRow, error)
	GetCustomersWithOrderStats(ctx context.Context, limit int32) ([]GetCustomersWithOrderStatsRow, error)
	GetCustomersWithoutStripeID(ctx context.Context, arg GetCustomersWithoutStripeIDParams) ([]Customers, error)
	GetLowStockProducts(ctx context.Context, stock int32) ([]Products, error)
	// internal/database/queries/orders.sql - Updated queries
	GetOrder(ctx context.Context, id int32) (Orders, error)
	GetOrderByStripeChargeID(ctx context.Context, stripeChargeID pgtype.Text) (Orders, error)
	GetOrderByStripePaymentIntentID(ctx context.Context, stripePaymentIntentID pgtype.Text) (Orders, error)
	GetOrderCountByStatus(ctx context.Context) ([]GetOrderCountByStatusRow, error)
	// internal/database/queries/order_items.sql - Updated queries
	GetOrderItem(ctx context.Context, id int32) (GetOrderItemRow, error)
	GetOrderItemStats(ctx context.Context, orderID int32) (GetOrderItemStatsRow, error)
	GetOrderItems(ctx context.Context, orderID int32) ([]GetOrderItemsRow, error)
	GetOrderItemsByProduct(ctx context.Context, arg GetOrderItemsByProductParams) ([]GetOrderItemsByProductRow, error)
	GetOrderItemsByPurchaseType(ctx context.Context, arg GetOrderItemsByPurchaseTypeParams) ([]GetOrderItemsByPurchaseTypeRow, error)
	GetOrderStats(ctx context.Context) (GetOrderStatsRow, error)
	GetOrdersByCustomerID(ctx context.Context, arg GetOrdersByCustomerIDParams) ([]Orders, error)
	GetOrdersByCustomerIDAndDateRange(ctx context.Context, arg GetOrdersByCustomerIDAndDateRangeParams) ([]Orders, error)
	GetOrdersByCustomerIDAndStatus(ctx context.Context, arg GetOrdersByCustomerIDAndStatusParams) ([]Orders, error)
	GetOrdersByCustomerIDWithStatusAndDateRange(ctx context.Context, arg GetOrdersByCustomerIDWithStatusAndDateRangeParams) ([]Orders, error)
	GetOrdersByStatus(ctx context.Context, arg GetOrdersByStatusParams) ([]Orders, error)
	// internal/database/queries/products.sql
	GetProduct(ctx context.Context, id int32) (Products, error)
	GetProductByName(ctx context.Context, name string) (Products, error)
	GetProductByStripeProductID(ctx context.Context, stripeProductID pgtype.Text) (Products, error)
	GetProductCount(ctx context.Context, active bool) (int64, error)
	GetProductsInStock(ctx context.Context) ([]Products, error)
	GetProductsWithoutStripeSync(ctx context.Context, arg GetProductsWithoutStripeSyncParams) ([]Products, error)
	GetRecentCustomers(ctx context.Context, limit int32) ([]GetRecentCustomersRow, error)
	GetSubscriptionOrderItems(ctx context.Context, orderID int32) ([]GetSubscriptionOrderItemsRow, error)
	GetTotalProductValue(ctx context.Context) (int32, error)
	IncrementProductStock(ctx context.Context, arg IncrementProductStockParams) (Products, error)
	ListActiveCustomers(ctx context.Context, arg ListActiveCustomersParams) ([]ListActiveCustomersRow, error)
	ListAllProducts(ctx context.Context, arg ListAllProductsParams) ([]Products, error)
	ListCustomers(ctx context.Context, arg ListCustomersParams) ([]ListCustomersRow, error)
	ListProducts(ctx context.Context) ([]Products, error)
	ListProductsByStatus(ctx context.Context, arg ListProductsByStatusParams) ([]Products, error)
	RestoreCustomer(ctx context.Context, id int32) (RestoreCustomerRow, error)
	SearchCustomers(ctx context.Context, arg SearchCustomersParams) ([]SearchCustomersRow, error)
	SearchCustomersByEmail(ctx context.Context, arg SearchCustomersByEmailParams) ([]SearchCustomersByEmailRow, error)
	SearchProducts(ctx context.Context, name string) ([]Products, error)
	UpdateCartItem(ctx context.Context, arg UpdateCartItemParams) (CartItems, error)
	UpdateCartItemQuantity(ctx context.Context, arg UpdateCartItemQuantityParams) (CartItems, error)
	UpdateCartTimestamp(ctx context.Context, id int32) (Carts, error)
	UpdateCustomer(ctx context.Context, arg UpdateCustomerParams) (UpdateCustomerRow, error)
	UpdateCustomerPassword(ctx context.Context, arg UpdateCustomerPasswordParams) (UpdateCustomerPasswordRow, error)
	UpdateCustomerStripeID(ctx context.Context, arg UpdateCustomerStripeIDParams) (Customers, error)
	UpdateOrderStatus(ctx context.Context, arg UpdateOrderStatusParams) (UpdateOrderStatusRow, error)
	UpdateProduct(ctx context.Context, arg UpdateProductParams) (Products, error)
	UpdateProductPrice(ctx context.Context, arg UpdateProductPriceParams) (Products, error)
	UpdateProductStock(ctx context.Context, arg UpdateProductStockParams) (Products, error)
	UpdateProductStripePrices(ctx context.Context, arg UpdateProductStripePricesParams) (Products, error)
	UpdateProductStripeProductID(ctx context.Context, arg UpdateProductStripeProductIDParams) (Products, error)
	UpdateStripeChargeID(ctx context.Context, arg UpdateStripeChargeIDParams) (UpdateStripeChargeIDRow, error)
}

var _ Querier = (*Queries)(nil)
