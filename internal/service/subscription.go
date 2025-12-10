package service

import (
	"github.com/dukerupert/hiri/internal/domain"
)

// SubscriptionService is re-exported from domain for backwards compatibility.
type SubscriptionService = domain.SubscriptionService

// Type aliases for backwards compatibility - all types now live in domain package.
type (
	SubscriptionCounts        = domain.SubscriptionCounts
	CreateSubscriptionParams  = domain.CreateSubscriptionParams
	GetSubscriptionParams     = domain.GetSubscriptionParams
	ListSubscriptionsParams   = domain.ListSubscriptionsParams
	PauseSubscriptionParams   = domain.PauseSubscriptionParams
	ResumeSubscriptionParams  = domain.ResumeSubscriptionParams
	CancelSubscriptionParams  = domain.CancelSubscriptionParams
	PortalSessionParams       = domain.PortalSessionParams
	SyncSubscriptionParams    = domain.SyncSubscriptionParams
	SubscriptionDetail        = domain.SubscriptionDetail
	SubscriptionSummary       = domain.SubscriptionSummary
	SubscriptionItemDetail    = domain.SubscriptionItemDetail
	AddressDetail             = domain.SubscriptionAddressDetail
	PaymentMethodDetail       = domain.SubscriptionPaymentMethodDetail
	UpcomingInvoiceDetail     = domain.UpcomingInvoiceDetail
)

// Billing interval constants re-exported from domain for backwards compatibility.
const (
	BillingIntervalWeekly       = domain.BillingIntervalWeekly
	BillingIntervalBiweekly     = domain.BillingIntervalBiweekly
	BillingIntervalMonthly      = domain.BillingIntervalMonthly
	BillingIntervalEvery6Weeks  = domain.BillingIntervalEvery6Weeks
	BillingIntervalEvery2Months = domain.BillingIntervalEvery2Months
)

// ValidBillingIntervals re-exported from domain for backwards compatibility.
var ValidBillingIntervals = domain.ValidBillingIntervals

// IsValidBillingInterval re-exported from domain for backwards compatibility.
var IsValidBillingInterval = domain.IsValidBillingInterval

// MapBillingIntervalToStripe re-exported from domain for backwards compatibility.
var MapBillingIntervalToStripe = domain.MapBillingIntervalToStripe
