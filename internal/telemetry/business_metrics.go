package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics holds Prometheus metrics for business-level observability.
// All metrics include tenant_id label for multi-tenant dashboard segmentation.
type BusinessMetrics struct {
	// Product engagement
	ProductViews      *prometheus.CounterVec
	ProductAddToCart  *prometheus.CounterVec
	ProductSearches   *prometheus.CounterVec
	SubscribePageView *prometheus.CounterVec

	// Checkout funnel
	CheckoutStarted   *prometheus.CounterVec
	CheckoutStep      *prometheus.CounterVec
	CheckoutCompleted *prometheus.CounterVec
	CheckoutAbandoned *prometheus.CounterVec
	PaymentAttempts   *prometheus.CounterVec
	PaymentSucceeded  *prometheus.CounterVec
	PaymentFailed     *prometheus.CounterVec

	// Orders
	OrdersCreated     *prometheus.CounterVec
	OrderValue        *prometheus.HistogramVec
	OrderItemCount    *prometheus.HistogramVec
	OrdersByType      *prometheus.CounterVec
	OrderFulfillment  *prometheus.HistogramVec

	// Subscriptions
	SubscriptionsCreated  *prometheus.CounterVec
	SubscriptionsCanceled *prometheus.CounterVec
	SubscriptionsPaused   *prometheus.CounterVec
	SubscriptionsResumed  *prometheus.CounterVec
	SubscriptionRenewals  *prometheus.CounterVec
	SubscriptionChurn     *prometheus.CounterVec

	// Cart
	CartCreated  *prometheus.CounterVec
	CartUpdated  *prometheus.CounterVec
	CartCleared  *prometheus.CounterVec
	CartValue    *prometheus.HistogramVec
	CartItemsAdd *prometheus.CounterVec

	// Webhooks
	WebhookReceived  *prometheus.CounterVec
	WebhookProcessed *prometheus.CounterVec
	WebhookFailed    *prometheus.CounterVec
	WebhookLatency   *prometheus.HistogramVec

	// Auth & accounts
	Signups          *prometheus.CounterVec
	Logins           *prometheus.CounterVec
	LoginFailed      *prometheus.CounterVec
	PasswordResets   *prometheus.CounterVec
	EmailVerified    *prometheus.CounterVec

	// Wholesale
	WholesaleApplications *prometheus.CounterVec

	// Background jobs
	JobsEnqueued   *prometheus.CounterVec
	JobsProcessed  *prometheus.CounterVec
	JobsFailed     *prometheus.CounterVec
	JobDuration    *prometheus.HistogramVec

	// Revenue tracking
	RevenueCollected *prometheus.CounterVec
	RefundsIssued    *prometheus.CounterVec
	RefundAmount     *prometheus.CounterVec

	// Email delivery
	EmailSent   *prometheus.CounterVec
	EmailFailed *prometheus.CounterVec

	// External API performance
	StripeAPILatency *prometheus.HistogramVec
}

// NewBusinessMetrics creates and registers all business metrics
func NewBusinessMetrics(namespace string) *BusinessMetrics {
	if namespace == "" {
		namespace = "hiri"
	}

	subsystem := "business"

	m := &BusinessMetrics{
		// =======================================================================
		// Product Engagement
		// =======================================================================
		ProductViews: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "product_views_total",
				Help:      "Total product detail page views",
			},
			[]string{"tenant_id", "product_slug"},
		),
		ProductAddToCart: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "product_add_to_cart_total",
				Help:      "Total add to cart actions",
			},
			[]string{"tenant_id", "product_id"},
		),
		ProductSearches: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "product_searches_total",
				Help:      "Total product list page views with filters",
			},
			[]string{"tenant_id", "filter_type"}, // filter_type: roast, origin, note, none
		),
		SubscribePageView: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscribe_page_views_total",
				Help:      "Total subscription product page views",
			},
			[]string{"tenant_id"},
		),

		// =======================================================================
		// Checkout Funnel
		// =======================================================================
		CheckoutStarted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "checkout_started_total",
				Help:      "Total checkout page loads",
			},
			[]string{"tenant_id"},
		),
		CheckoutStep: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "checkout_step_total",
				Help:      "Total completions of each checkout step",
			},
			[]string{"tenant_id", "step"}, // step: address, shipping, payment
		),
		CheckoutCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "checkout_completed_total",
				Help:      "Total successful checkouts",
			},
			[]string{"tenant_id"},
		),
		CheckoutAbandoned: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "checkout_abandoned_total",
				Help:      "Total abandoned checkouts (payment intent canceled)",
			},
			[]string{"tenant_id"},
		),
		PaymentAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "payment_attempts_total",
				Help:      "Total payment attempts",
			},
			[]string{"tenant_id", "payment_type"}, // payment_type: one_time, subscription
		),
		PaymentSucceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "payment_succeeded_total",
				Help:      "Total successful payments",
			},
			[]string{"tenant_id", "payment_type"},
		),
		PaymentFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "payment_failed_total",
				Help:      "Total failed payments",
			},
			[]string{"tenant_id", "payment_type", "failure_reason"},
		),

		// =======================================================================
		// Orders
		// =======================================================================
		OrdersCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "orders_created_total",
				Help:      "Total orders created",
			},
			[]string{"tenant_id", "order_type"}, // order_type: one_time, subscription
		),
		OrderValue: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "order_value_cents",
				Help:      "Order value distribution in cents",
				Buckets:   []float64{1000, 2500, 5000, 7500, 10000, 15000, 25000, 50000, 100000},
			},
			[]string{"tenant_id", "order_type"},
		),
		OrderItemCount: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "order_item_count",
				Help:      "Number of items per order",
				Buckets:   []float64{1, 2, 3, 5, 10, 15, 20},
			},
			[]string{"tenant_id"},
		),
		OrdersByType: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "orders_by_type_total",
				Help:      "Total orders by customer type",
			},
			[]string{"tenant_id", "customer_type"}, // customer_type: retail, wholesale
		),
		OrderFulfillment: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "order_fulfillment_seconds",
				Help:      "Time from order creation to fulfillment",
				Buckets:   []float64{3600, 14400, 28800, 43200, 86400, 172800, 259200, 604800}, // 1h to 7d
			},
			[]string{"tenant_id"},
		),

		// =======================================================================
		// Subscriptions
		// =======================================================================
		SubscriptionsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscriptions_created_total",
				Help:      "Total subscriptions created",
			},
			[]string{"tenant_id", "billing_interval"}, // billing_interval: weekly, biweekly, monthly, etc.
		),
		SubscriptionsCanceled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscriptions_canceled_total",
				Help:      "Total subscriptions canceled",
			},
			[]string{"tenant_id", "cancel_reason"},
		),
		SubscriptionsPaused: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscriptions_paused_total",
				Help:      "Total subscriptions paused",
			},
			[]string{"tenant_id"},
		),
		SubscriptionsResumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscriptions_resumed_total",
				Help:      "Total subscriptions resumed from pause",
			},
			[]string{"tenant_id"},
		),
		SubscriptionRenewals: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscription_renewals_total",
				Help:      "Total successful subscription renewals",
			},
			[]string{"tenant_id"},
		),
		SubscriptionChurn: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "subscription_churn_total",
				Help:      "Total subscriptions churned (expired, payment failed permanently)",
			},
			[]string{"tenant_id", "churn_reason"}, // churn_reason: payment_failed, canceled, expired
		),

		// =======================================================================
		// Cart
		// =======================================================================
		CartCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cart_created_total",
				Help:      "Total carts created",
			},
			[]string{"tenant_id"},
		),
		CartUpdated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cart_updated_total",
				Help:      "Total cart update operations",
			},
			[]string{"tenant_id", "action"}, // action: add, remove, update_quantity
		),
		CartCleared: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cart_cleared_total",
				Help:      "Total carts cleared (after purchase or manually)",
			},
			[]string{"tenant_id", "reason"}, // reason: purchase, manual
		),
		CartValue: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cart_value_cents",
				Help:      "Cart value at checkout start",
				Buckets:   []float64{1000, 2500, 5000, 7500, 10000, 15000, 25000, 50000},
			},
			[]string{"tenant_id"},
		),
		CartItemsAdd: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cart_items_added_total",
				Help:      "Total items added to carts (quantity-aware)",
			},
			[]string{"tenant_id"},
		),

		// =======================================================================
		// Webhooks (Stripe)
		// =======================================================================
		WebhookReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "webhook_received_total",
				Help:      "Total webhooks received",
			},
			[]string{"tenant_id", "event_type"},
		),
		WebhookProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "webhook_processed_total",
				Help:      "Total webhooks successfully processed",
			},
			[]string{"tenant_id", "event_type"},
		),
		WebhookFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "webhook_failed_total",
				Help:      "Total webhook processing failures",
			},
			[]string{"tenant_id", "event_type", "error_type"},
		),
		WebhookLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "webhook_processing_seconds",
				Help:      "Webhook processing duration",
				Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"tenant_id", "event_type"},
		),

		// =======================================================================
		// Auth & Accounts
		// =======================================================================
		Signups: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "signups_total",
				Help:      "Total successful user signups",
			},
			[]string{"tenant_id"},
		),
		Logins: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "logins_total",
				Help:      "Total successful logins",
			},
			[]string{"tenant_id", "user_type"}, // user_type: customer, admin
		),
		LoginFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "login_failed_total",
				Help:      "Total failed login attempts",
			},
			[]string{"tenant_id", "reason"}, // reason: invalid_password, user_not_found, account_locked
		),
		PasswordResets: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "password_resets_total",
				Help:      "Total password reset requests",
			},
			[]string{"tenant_id"},
		),
		EmailVerified: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "email_verified_total",
				Help:      "Total email verifications completed",
			},
			[]string{"tenant_id"},
		),

		// =======================================================================
		// Wholesale
		// =======================================================================
		WholesaleApplications: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "wholesale_applications_total",
				Help:      "Total wholesale applications submitted",
			},
			[]string{"tenant_id", "status"}, // status: submitted, approved, rejected
		),

		// =======================================================================
		// Background Jobs
		// =======================================================================
		JobsEnqueued: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "jobs_enqueued_total",
				Help:      "Total background jobs enqueued",
			},
			[]string{"tenant_id", "job_type"},
		),
		JobsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "jobs_processed_total",
				Help:      "Total background jobs successfully processed",
			},
			[]string{"tenant_id", "job_type"},
		),
		JobsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "jobs_failed_total",
				Help:      "Total background job failures",
			},
			[]string{"tenant_id", "job_type", "error_type"},
		),
		JobDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "job_duration_seconds",
				Help:      "Background job execution duration",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"tenant_id", "job_type"},
		),

		// =======================================================================
		// Revenue Tracking
		// =======================================================================
		RevenueCollected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "revenue_collected_cents",
				Help:      "Total revenue collected in cents (excludes refunds)",
			},
			[]string{"tenant_id", "order_type"}, // order_type: one_time, subscription
		),
		RefundsIssued: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "refunds_issued_total",
				Help:      "Total refunds issued to customers",
			},
			[]string{"tenant_id", "reason"}, // reason: customer_request, fraudulent, duplicate, etc.
		),
		RefundAmount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "refund_amount_cents",
				Help:      "Total refund amount in cents",
			},
			[]string{"tenant_id"},
		),

		// =======================================================================
		// Email Delivery
		// =======================================================================
		EmailSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "emails_sent_total",
				Help:      "Total emails sent by type",
			},
			[]string{"tenant_id", "email_type"}, // email_type: order_confirmation, subscription_renewal, password_reset, etc.
		),
		EmailFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "emails_failed_total",
				Help:      "Total email delivery failures",
			},
			[]string{"tenant_id", "email_type", "error_type"},
		),

		// =======================================================================
		// External API Performance
		// =======================================================================
		StripeAPILatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "stripe_api_duration_seconds",
				Help:      "Stripe API call duration (helps differentiate app slowness from Stripe issues)",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"tenant_id", "operation"}, // operation: create_payment_intent, create_subscription, create_customer, etc.
		),
	}

	return m
}

// Global instance for easy access from handlers
var Business *BusinessMetrics

// InitBusinessMetrics initializes the global business metrics instance
func InitBusinessMetrics(namespace string) *BusinessMetrics {
	Business = NewBusinessMetrics(namespace)
	return Business
}
