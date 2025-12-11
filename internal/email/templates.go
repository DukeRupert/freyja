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

// EmailVerificationEmail represents an email verification email
type EmailVerificationEmail struct {
	Email     string
	FirstName string
	VerifyURL string
	ExpiresAt time.Time
}

func (e EmailVerificationEmail) Subject() string {
	return "Verify Your Email Address"
}

func (e EmailVerificationEmail) TemplateName() string {
	return "email_verification.html"
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

// InvoiceSentEmail represents an invoice sent notification email
type InvoiceSentEmail struct {
	Email         string
	CustomerName  string
	InvoiceNumber string
	InvoiceDate   time.Time
	DueDate       time.Time
	PaymentTerms  string
	Items         []InvoiceItem
	SubtotalCents int64
	ShippingCents int64
	TaxCents      int64
	DiscountCents int64
	TotalCents    int64
	PaymentURL    string
}

func (e InvoiceSentEmail) Subject() string {
	return "Invoice " + e.InvoiceNumber + " from Hiri"
}

func (e InvoiceSentEmail) TemplateName() string {
	return "invoice_sent.html"
}

// InvoiceReminderEmail represents an invoice payment reminder email
type InvoiceReminderEmail struct {
	Email         string
	CustomerName  string
	InvoiceNumber string
	DueDate       time.Time
	BalanceCents  int64
	ReminderType  string // "approaching_due" or "past_due"
	DaysBefore    int    // Days before due date (for approaching_due)
	DaysOverdue   int    // Days past due date (for past_due)
	PaymentURL    string
}

func (e InvoiceReminderEmail) Subject() string {
	if e.ReminderType == "past_due" {
		return "Payment Reminder - Invoice " + e.InvoiceNumber + " Past Due"
	}
	return "Payment Reminder - Invoice " + e.InvoiceNumber
}

func (e InvoiceReminderEmail) TemplateName() string {
	return "invoice_reminder.html"
}

// InvoiceOverdueEmail represents an invoice overdue notification email
type InvoiceOverdueEmail struct {
	Email         string
	CustomerName  string
	InvoiceNumber string
	DueDate       time.Time
	BalanceCents  int64
	DaysOverdue   int
	PaymentURL    string
}

func (e InvoiceOverdueEmail) Subject() string {
	return "Invoice " + e.InvoiceNumber + " is Overdue"
}

func (e InvoiceOverdueEmail) TemplateName() string {
	return "invoice_overdue.html"
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

// InvoiceItem represents a line item on an invoice
type InvoiceItem struct {
	Description string
	Quantity    int
	UnitCents   int64
	TotalCents  int64
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

// SaaS Platform Emails (for tenant operators)

// OperatorSetupEmail represents an operator account setup email
type OperatorSetupEmail struct {
	Email     string
	Name      string
	SetupURL  string
	ExpiresAt time.Time
}

func (e OperatorSetupEmail) Subject() string {
	return "Complete Your Hiri Account Setup"
}

func (e OperatorSetupEmail) TemplateName() string {
	return "operator_setup.html"
}

// OperatorPasswordResetEmail represents an operator password reset email
type OperatorPasswordResetEmail struct {
	Email     string
	Name      string
	ResetURL  string
	ExpiresAt time.Time
}

func (e OperatorPasswordResetEmail) Subject() string {
	return "Reset Your Hiri Password"
}

func (e OperatorPasswordResetEmail) TemplateName() string {
	return "operator_password_reset.html"
}

// PlatformPaymentFailedEmail represents a platform subscription payment failure email
type PlatformPaymentFailedEmail struct {
	Email            string
	Name             string
	UpdatePaymentURL string
}

func (e PlatformPaymentFailedEmail) Subject() string {
	return "Payment Issue with Your Hiri Subscription"
}

func (e PlatformPaymentFailedEmail) TemplateName() string {
	return "platform_payment_failed.html"
}

// PlatformSuspendedEmail represents a platform subscription suspended email
type PlatformSuspendedEmail struct {
	Email            string
	Name             string
	UpdatePaymentURL string
}

func (e PlatformSuspendedEmail) Subject() string {
	return "Your Hiri Store Has Been Suspended"
}

func (e PlatformSuspendedEmail) TemplateName() string {
	return "platform_suspended.html"
}

// Wholesale Application Emails

// WholesaleApprovedEmail represents a wholesale application approved email
type WholesaleApprovedEmail struct {
	Email        string
	CustomerName string
	LoginURL     string
}

func (e WholesaleApprovedEmail) Subject() string {
	return "Your Wholesale Account Has Been Approved!"
}

func (e WholesaleApprovedEmail) TemplateName() string {
	return "wholesale_approved.html"
}

// WholesaleRejectedEmail represents a wholesale application rejected email
type WholesaleRejectedEmail struct {
	Email           string
	CustomerName    string
	RejectionReason string
	ShopURL         string
}

func (e WholesaleRejectedEmail) Subject() string {
	return "Your Wholesale Application Status"
}

func (e WholesaleRejectedEmail) TemplateName() string {
	return "wholesale_rejected.html"
}
