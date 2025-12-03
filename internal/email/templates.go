package email

import "time"

// EmailTemplate defines the interface for email templates
type EmailTemplate interface {
	Subject() string
	TemplateName() string
}

// PasswordResetEmail represents a password reset email
type PasswordResetEmail struct {
	Email     string
	FirstName string
	ResetURL  string
	ExpiresAt time.Time
}

func (e PasswordResetEmail) Subject() string {
	return "Reset Your Password"
}

func (e PasswordResetEmail) TemplateName() string {
	return "password_reset.html"
}

// OrderConfirmationEmail represents an order confirmation email
type OrderConfirmationEmail struct {
	OrderNumber   string
	CustomerName  string
	OrderDate     time.Time
	Items         []OrderItem
	SubtotalCents int64
	ShippingCents int64
	TaxCents      int64
	TotalCents    int64
	ShippingAddr  Address
	BillingAddr   Address
	TrackingURL   string // Optional, may be empty at order creation
}

func (e OrderConfirmationEmail) Subject() string {
	return "Order Confirmation - " + e.OrderNumber
}

func (e OrderConfirmationEmail) TemplateName() string {
	return "order_confirmation.html"
}

// ShippingConfirmationEmail represents a shipping confirmation email
type ShippingConfirmationEmail struct {
	OrderNumber    string
	CustomerName   string
	ShippedDate    time.Time
	Items          []OrderItem
	ShippingAddr   Address
	Carrier        string
	TrackingNumber string
	TrackingURL    string
}

func (e ShippingConfirmationEmail) Subject() string {
	return "Your Order Has Shipped - " + e.OrderNumber
}

func (e ShippingConfirmationEmail) TemplateName() string {
	return "shipping_confirmation.html"
}

// SubscriptionWelcomeEmail represents a subscription welcome email
type SubscriptionWelcomeEmail struct {
	CustomerName      string
	ProductName       string
	Frequency         string // e.g., "Every 2 weeks"
	NextDeliveryDate  time.Time
	ManagementURL     string
	ShippingAddr      Address
	SubscriptionTotal int64 // In cents
}

func (e SubscriptionWelcomeEmail) Subject() string {
	return "Welcome to Your Coffee Subscription"
}

func (e SubscriptionWelcomeEmail) TemplateName() string {
	return "subscription_welcome.html"
}

// SubscriptionPaymentFailedEmail represents a failed subscription payment email
type SubscriptionPaymentFailedEmail struct {
	CustomerName     string
	ProductName      string
	FailedDate       time.Time
	RetryDate        time.Time
	UpdatePaymentURL string
	ManagementURL    string
}

func (e SubscriptionPaymentFailedEmail) Subject() string {
	return "Subscription Payment Issue"
}

func (e SubscriptionPaymentFailedEmail) TemplateName() string {
	return "subscription_payment_failed.html"
}

// SubscriptionCancelledEmail represents a subscription cancellation email
type SubscriptionCancelledEmail struct {
	CustomerName      string
	ProductName       string
	CancelledDate     time.Time
	FinalDeliveryDate time.Time // If there's a final shipment
	HasFinalDelivery  bool
	ReactivationURL   string
}

func (e SubscriptionCancelledEmail) Subject() string {
	return "Subscription Cancelled"
}

func (e SubscriptionCancelledEmail) TemplateName() string {
	return "subscription_cancelled.html"
}

// Supporting types

// OrderItem represents a line item in an order
type OrderItem struct {
	ProductName string
	VariantName string // e.g., "12oz, Whole Bean"
	Quantity    int
	PriceCents  int64
	TotalCents  int64
	ImageURL    string // Optional product image
}

// Address represents a shipping or billing address
type Address struct {
	Name       string
	Company    string // Optional
	Line1      string
	Line2      string // Optional
	City       string
	State      string
	PostalCode string
	Country    string
}
