package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// subscriptionService implements SubscriptionService interface
type subscriptionService struct {
	repo            repository.Querier
	tenantID        pgtype.UUID
	billingProvider billing.Provider
}

// NewSubscriptionService creates a new SubscriptionService instance
func NewSubscriptionService(repo repository.Querier, tenantID string, billingProvider billing.Provider) (SubscriptionService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &subscriptionService{
		repo:            repo,
		tenantID:        tenantUUID,
		billingProvider: billingProvider,
	}, nil
}

// CreateSubscription creates a new subscription for a customer.
//
// Flow:
//  1. Validate billing interval
//  2. Get product SKU and pricing
//  3. Get billing customer (Stripe customer ID)
//  4. Get payment method
//  5. Calculate pricing (subtotal, shipping, tax, total)
//  6. Create local subscription record (pending status)
//  7. Create Stripe recurring price for this subscription
//  8. Create Stripe subscription
//  9. Update local subscription with Stripe subscription ID
// 10. Create subscription item record
// 11. Create schedule event for tracking
// 12. Return subscription details
func (s *subscriptionService) CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*SubscriptionDetail, error) {
	// Step 1: Validate billing interval
	if !IsValidBillingInterval(params.BillingInterval) {
		return nil, ErrInvalidBillingInterval
	}

	// Step 2: Get product SKU
	sku, err := s.repo.GetSKUByID(ctx, params.ProductSKUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSKUNotFound
		}
		return nil, fmt.Errorf("failed to get product SKU: %w", err)
	}

	// Get default price list for tenant
	defaultPriceList, err := s.repo.GetDefaultPriceList(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default price list: %w", err)
	}

	// Get price for SKU from default price list
	priceEntry, err := s.repo.GetPriceForSKU(ctx, repository.GetPriceForSKUParams{
		PriceListID:  defaultPriceList.ID,
		ProductSkuID: params.ProductSKUID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPriceNotFound
		}
		return nil, fmt.Errorf("failed to get SKU price: %w", err)
	}

	// Step 3: Get billing customer for user
	billingCustomer, err := s.repo.GetBillingCustomerForUser(ctx, repository.GetBillingCustomerForUserParams{
		UserID:   params.UserID,
		TenantID: s.tenantID,
		Provider: "stripe",
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoPaymentMethod
		}
		return nil, fmt.Errorf("failed to get billing customer: %w", err)
	}

	// Step 4: Get payment method
	paymentMethod, err := s.repo.GetPaymentMethodByID(ctx, repository.GetPaymentMethodByIDParams{
		ID:       params.PaymentMethodID,
		TenantID: s.tenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoPaymentMethod
		}
		return nil, fmt.Errorf("failed to get payment method: %w", err)
	}

	// Validate payment method belongs to billing customer
	if paymentMethod.BillingCustomerID.Bytes != billingCustomer.ID.Bytes {
		return nil, fmt.Errorf("payment method does not belong to user")
	}

	// Step 5: Calculate pricing
	unitPriceCents := priceEntry.PriceCents
	subtotalCents := unitPriceCents * params.Quantity
	shippingCents := int32(0) // TODO: Calculate shipping based on shipping_method_id
	taxCents := int32(0)      // TODO: Calculate tax if needed
	totalCents := subtotalCents + shippingCents + taxCents
	currency := "usd" // Default currency, could be from price list in future

	// Step 6: Create local subscription record (pending status)
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	currentPeriodEnd := pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true} // Placeholder

	metadata, _ := json.Marshal(map[string]string{
		"user_id":          uuidToString(params.UserID),
		"product_sku_id":   uuidToString(params.ProductSKUID),
		"billing_interval": params.BillingInterval,
	})

	subscription, err := s.repo.CreateSubscription(ctx, repository.CreateSubscriptionParams{
		TenantID:               s.tenantID,
		UserID:                 params.UserID,
		SubscriptionPlanID:     pgtype.UUID{}, // NULL for custom subscriptions
		BillingInterval:        params.BillingInterval,
		Status:                 "pending",
		BillingCustomerID:      billingCustomer.ID,
		Provider:               "stripe",
		ProviderSubscriptionID: pgtype.Text{}, // Will update after Stripe creation
		SubtotalCents:          subtotalCents,
		TaxCents:               taxCents,
		TotalCents:             totalCents,
		Currency:               currency,
		ShippingAddressID:      params.ShippingAddressID,
		ShippingMethodID:       params.ShippingMethodID,
		ShippingCents:          shippingCents,
		PaymentMethodID:        params.PaymentMethodID,
		CurrentPeriodStart:     now,
		CurrentPeriodEnd:       currentPeriodEnd,
		NextBillingDate:        currentPeriodEnd,
		Metadata:               metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription record: %w", err)
	}

	// Step 7: Get product name for Stripe Product creation
	product, err := s.repo.GetProductByID(ctx, repository.GetProductByIDParams{
		ID:       sku.ProductID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	// Step 8: Map billing interval to Stripe format
	stripeInterval, intervalCount, err := MapBillingIntervalToStripe(params.BillingInterval)
	if err != nil {
		return nil, err
	}

	// Step 9: Create or get Stripe Product for this SKU
	// Product name includes weight and grind for uniqueness
	weightStr := ""
	if sku.WeightValue.Valid {
		weightStr = sku.WeightValue.Int.String()
	}
	productName := fmt.Sprintf("%s - %s %s", product.Name, weightStr, sku.WeightUnit)
	if sku.Grind != "" && sku.Grind != "whole_bean" {
		productName = fmt.Sprintf("%s (%s)", productName, sku.Grind)
	}

	stripeProduct, err := s.billingProvider.CreateProduct(ctx, billing.CreateProductParams{
		Name:        productName,
		Description: product.Description.String,
		Active:      true,
		Metadata: map[string]string{
			"tenant_id":      uuidToString(s.tenantID),
			"product_id":     uuidToString(product.ID),
			"product_sku_id": uuidToString(params.ProductSKUID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe product: %w", err)
	}

	priceNickname := fmt.Sprintf("%s - %s", sku.Sku, params.BillingInterval)

	stripePrice, err := s.billingProvider.CreateRecurringPrice(ctx, billing.CreateRecurringPriceParams{
		Currency:        currency,
		UnitAmountCents: unitPriceCents,
		BillingInterval: stripeInterval,
		IntervalCount:   intervalCount,
		ProductID:       stripeProduct.ID,
		Metadata: map[string]string{
			"tenant_id":        uuidToString(s.tenantID),
			"subscription_id":  uuidToString(subscription.ID),
			"product_sku_id":   uuidToString(params.ProductSKUID),
			"billing_interval": params.BillingInterval,
		},
		Nickname: priceNickname,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe price: %w", err)
	}

	// Step 9: Create Stripe subscription
	stripeSubscription, err := s.billingProvider.CreateSubscription(ctx, billing.CreateSubscriptionParams{
		TenantID:               uuidToString(s.tenantID),
		CustomerID:             billingCustomer.ProviderCustomerID,
		PriceID:                stripePrice.ID,
		Quantity:               params.Quantity,
		DefaultPaymentMethodID: paymentMethod.ProviderPaymentMethodID,
		CollectionMethod:       "charge_automatically",
		Metadata: map[string]string{
			"tenant_id":        uuidToString(s.tenantID),
			"subscription_id":  uuidToString(subscription.ID),
			"user_id":          uuidToString(params.UserID),
			"product_sku_id":   uuidToString(params.ProductSKUID),
			"billing_interval": params.BillingInterval,
		},
		IdempotencyKey: fmt.Sprintf("sub_%s", uuidToString(subscription.ID)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	// Step 10: Update local subscription with Stripe subscription ID and dates
	subscription, err = s.repo.UpdateSubscriptionProviderID(ctx, repository.UpdateSubscriptionProviderIDParams{
		ID:                     subscription.ID,
		TenantID:               s.tenantID,
		ProviderSubscriptionID: pgtype.Text{String: stripeSubscription.ID, Valid: true},
		Status:                 stripeSubscription.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription with provider ID: %w", err)
	}

	// Update subscription dates from Stripe
	subscription, err = s.repo.UpdateSubscriptionStatus(ctx, repository.UpdateSubscriptionStatusParams{
		ID:                 subscription.ID,
		TenantID:           s.tenantID,
		Status:             stripeSubscription.Status,
		CurrentPeriodStart: pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodStart, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodEnd, Valid: true},
		NextBillingDate:    pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodEnd, Valid: true},
		CancelAtPeriodEnd:  pgtype.Bool{Bool: stripeSubscription.CancelAtPeriodEnd, Valid: true},
		CancelledAt:        pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription dates: %w", err)
	}

	// Step 11: Create subscription item record
	itemMetadata, _ := json.Marshal(map[string]string{
		"sku":   sku.Sku,
		"grind": sku.Grind,
	})

	_, err = s.repo.CreateSubscriptionItem(ctx, repository.CreateSubscriptionItemParams{
		TenantID:       s.tenantID,
		SubscriptionID: subscription.ID,
		ProductSkuID:   params.ProductSKUID,
		Quantity:       params.Quantity,
		UnitPriceCents: unitPriceCents,
		Metadata:       itemMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription item: %w", err)
	}

	// Step 12: Create schedule event for creation
	scheduleMetadata, _ := json.Marshal(map[string]string{
		"event":                 "subscription_created",
		"stripe_subscription_id": stripeSubscription.ID,
	})

	_, err = s.repo.CreateSubscriptionScheduleEvent(ctx, repository.CreateSubscriptionScheduleEventParams{
		TenantID:       s.tenantID,
		SubscriptionID: subscription.ID,
		EventType:      "billing",
		Status:         "completed",
		ScheduledAt:    now,
		OrderID:        pgtype.UUID{}, // No order yet (first invoice will create order)
		PaymentID:      pgtype.UUID{},
		Metadata:       scheduleMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule event: %w", err)
	}

	// Step 13: Return subscription details
	return s.GetSubscription(ctx, GetSubscriptionParams{
		TenantID:               s.tenantID,
		SubscriptionID:         subscription.ID,
		IncludeUpcomingInvoice: false,
	})
}

// GetSubscription retrieves subscription details
func (s *subscriptionService) GetSubscription(ctx context.Context, params GetSubscriptionParams) (*SubscriptionDetail, error) {
	// Get subscription with details (includes user, address, payment method)
	subWithDetails, err := s.repo.GetSubscriptionWithDetails(ctx, repository.GetSubscriptionWithDetailsParams{
		ID:       params.SubscriptionID,
		TenantID: params.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get subscription items
	items, err := s.repo.ListSubscriptionItemsForSubscription(ctx, repository.ListSubscriptionItemsForSubscriptionParams{
		SubscriptionID: params.SubscriptionID,
		TenantID:       params.TenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription items: %w", err)
	}

	// Build subscription detail
	detail := &SubscriptionDetail{
		ID:                     subWithDetails.ID,
		TenantID:               subWithDetails.TenantID,
		UserID:                 subWithDetails.UserID,
		Status:                 subWithDetails.Status,
		BillingInterval:        subWithDetails.BillingInterval,
		SubtotalCents:          subWithDetails.SubtotalCents,
		TaxCents:               subWithDetails.TaxCents,
		ShippingCents:          subWithDetails.ShippingCents,
		TotalCents:             subWithDetails.TotalCents,
		Currency:               subWithDetails.Currency,
		ProviderSubscriptionID: subWithDetails.ProviderSubscriptionID.String,
		ProviderCustomerID:     subWithDetails.ProviderCustomerID.String,
		Provider:               subWithDetails.Provider,
		CurrentPeriodStart:     subWithDetails.CurrentPeriodStart.Time,
		CurrentPeriodEnd:       subWithDetails.CurrentPeriodEnd.Time,
		NextBillingDate:        subWithDetails.NextBillingDate.Time,
		CancelAtPeriodEnd:      subWithDetails.CancelAtPeriodEnd,
		CreatedAt:              subWithDetails.CreatedAt.Time,
		UpdatedAt:              subWithDetails.UpdatedAt.Time,
		Items:                  make([]SubscriptionItemDetail, len(items)),
	}

	if subWithDetails.CancelledAt.Valid {
		cancelledAt := subWithDetails.CancelledAt.Time
		detail.CancelledAt = &cancelledAt
	}

	// Map subscription items
	for i, item := range items {
		weightValue := ""
		if item.WeightValue.Valid {
			weightValue = item.WeightValue.Int.String()
		}

		imageURL := ""
		if item.ProductImageUrl.Valid {
			imageURL = item.ProductImageUrl.String
		}

		detail.Items[i] = SubscriptionItemDetail{
			ID:             item.ID,
			ProductSKUID:   item.ProductSkuID,
			ProductName:    item.ProductName,
			SKU:            item.Sku,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
			ImageURL:       imageURL,
			WeightValue:    weightValue,
			WeightUnit:     item.WeightUnit,
			Grind:          item.Grind,
		}
	}

	// Add shipping address
	detail.ShippingAddress = &AddressDetail{
		ID:           subWithDetails.ShippingAddressID,
		FullName:     subWithDetails.ShippingFullName.String,
		Company:      subWithDetails.ShippingCompany.String,
		AddressLine1: subWithDetails.ShippingAddressLine1,
		AddressLine2: subWithDetails.ShippingAddressLine2.String,
		City:         subWithDetails.ShippingCity,
		State:        subWithDetails.ShippingState,
		PostalCode:   subWithDetails.ShippingPostalCode,
		Country:      subWithDetails.ShippingCountry,
		Phone:        subWithDetails.ShippingPhone.String,
	}

	// Add payment method if exists
	if subWithDetails.PaymentMethodType.Valid {
		detail.PaymentMethod = &PaymentMethodDetail{
			ID:              subWithDetails.PaymentMethodID,
			MethodType:      subWithDetails.PaymentMethodType.String,
			DisplayBrand:    subWithDetails.PaymentDisplayBrand.String,
			DisplayLast4:    subWithDetails.PaymentDisplayLast4.String,
			DisplayExpMonth: subWithDetails.PaymentDisplayExpMonth.Int32,
			DisplayExpYear:  subWithDetails.PaymentDisplayExpYear.Int32,
		}
	}

	return detail, nil
}

// ListSubscriptionsForUser retrieves all subscriptions for a customer
func (s *subscriptionService) ListSubscriptionsForUser(ctx context.Context, params ListSubscriptionsParams) ([]SubscriptionSummary, error) {
	// Use summary query for efficient listing
	summaries, err := s.repo.GetSubscriptionSummariesForUser(ctx, repository.GetSubscriptionSummariesForUserParams{
		UserID:   params.UserID,
		TenantID: params.TenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	result := make([]SubscriptionSummary, len(summaries))
	for i, summary := range summaries {
		imageURL := ""
		if summary.ProductImageUrl.Valid {
			imageURL = summary.ProductImageUrl.String
		}

		result[i] = SubscriptionSummary{
			ID:                summary.ID,
			Status:            summary.Status,
			BillingInterval:   summary.BillingInterval,
			TotalCents:        summary.TotalCents,
			Currency:          summary.Currency,
			NextBillingDate:   summary.NextBillingDate.Time,
			CancelAtPeriodEnd: summary.CancelAtPeriodEnd,
			ProductName:       summary.ProductName,
			ProductImageURL:   imageURL,
			CreatedAt:         summary.CreatedAt.Time,
		}
	}

	return result, nil
}

// PauseSubscription pauses a subscription until manually resumed
func (s *subscriptionService) PauseSubscription(ctx context.Context, params PauseSubscriptionParams) (*SubscriptionDetail, error) {
	// Get subscription
	subscription, err := s.repo.GetSubscriptionByID(ctx, repository.GetSubscriptionByIDParams{
		ID:       params.SubscriptionID,
		TenantID: params.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate subscription can be paused
	if subscription.Status != "active" {
		return nil, ErrSubscriptionNotActive
	}

	// Pause in Stripe
	_, err = s.billingProvider.PauseSubscription(ctx, billing.PauseSubscriptionParams{
		SubscriptionID: subscription.ProviderSubscriptionID.String,
		TenantID:       uuidToString(params.TenantID),
		Behavior:       "void", // Void pending invoices
		ResumesAt:      params.ResumesAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pause Stripe subscription: %w", err)
	}

	// Update local status
	_, err = s.repo.UpdateSubscriptionPauseResume(ctx, repository.UpdateSubscriptionPauseResumeParams{
		ID:       params.SubscriptionID,
		TenantID: params.TenantID,
		Status:   "paused",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription status: %w", err)
	}

	// Create schedule event
	scheduleMetadata, _ := json.Marshal(map[string]string{
		"event":      "paused",
		"resumes_at": formatTimePtr(params.ResumesAt),
	})

	_, err = s.repo.CreateSubscriptionScheduleEvent(ctx, repository.CreateSubscriptionScheduleEventParams{
		TenantID:       params.TenantID,
		SubscriptionID: params.SubscriptionID,
		EventType:      "pause",
		Status:         "completed",
		ScheduledAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OrderID:        pgtype.UUID{},
		PaymentID:      pgtype.UUID{},
		Metadata:       scheduleMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule event: %w", err)
	}

	// Return updated subscription
	return s.GetSubscription(ctx, GetSubscriptionParams{
		TenantID:               params.TenantID,
		SubscriptionID:         params.SubscriptionID,
		IncludeUpcomingInvoice: false,
	})
}

// ResumeSubscription resumes a paused subscription immediately
func (s *subscriptionService) ResumeSubscription(ctx context.Context, params ResumeSubscriptionParams) (*SubscriptionDetail, error) {
	// Get subscription
	subscription, err := s.repo.GetSubscriptionByID(ctx, repository.GetSubscriptionByIDParams{
		ID:       params.SubscriptionID,
		TenantID: params.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate subscription is paused
	if subscription.Status != "paused" {
		return nil, ErrSubscriptionNotPaused
	}

	// Resume in Stripe
	stripeSubscription, err := s.billingProvider.ResumeSubscription(ctx, billing.ResumeSubscriptionParams{
		SubscriptionID: subscription.ProviderSubscriptionID.String,
		TenantID:       uuidToString(params.TenantID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resume Stripe subscription: %w", err)
	}

	// Update local status
	_, err = s.repo.UpdateSubscriptionStatus(ctx, repository.UpdateSubscriptionStatusParams{
		ID:                 params.SubscriptionID,
		TenantID:           params.TenantID,
		Status:             stripeSubscription.Status,
		CurrentPeriodStart: pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodStart, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodEnd, Valid: true},
		NextBillingDate:    pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodEnd, Valid: true},
		CancelAtPeriodEnd:  pgtype.Bool{Bool: false, Valid: true},
		CancelledAt:        pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription status: %w", err)
	}

	// Create schedule event
	scheduleMetadata, _ := json.Marshal(map[string]string{
		"event": "resumed",
	})

	_, err = s.repo.CreateSubscriptionScheduleEvent(ctx, repository.CreateSubscriptionScheduleEventParams{
		TenantID:       params.TenantID,
		SubscriptionID: params.SubscriptionID,
		EventType:      "resume",
		Status:         "completed",
		ScheduledAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OrderID:        pgtype.UUID{},
		PaymentID:      pgtype.UUID{},
		Metadata:       scheduleMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule event: %w", err)
	}

	// Return updated subscription
	return s.GetSubscription(ctx, GetSubscriptionParams{
		TenantID:               params.TenantID,
		SubscriptionID:         params.SubscriptionID,
		IncludeUpcomingInvoice: false,
	})
}

// CancelSubscription cancels a subscription
func (s *subscriptionService) CancelSubscription(ctx context.Context, params CancelSubscriptionParams) (*SubscriptionDetail, error) {
	// Get subscription
	subscription, err := s.repo.GetSubscriptionByID(ctx, repository.GetSubscriptionByIDParams{
		ID:       params.SubscriptionID,
		TenantID: params.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Cancel in Stripe
	err = s.billingProvider.CancelSubscription(ctx, billing.CancelSubscriptionParams{
		SubscriptionID:     subscription.ProviderSubscriptionID.String,
		TenantID:           uuidToString(params.TenantID),
		CancelAtPeriodEnd:  params.CancelAtPeriodEnd,
		CancellationReason: params.CancellationReason,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	// Update local subscription
	cancellationReason := pgtype.Text{}
	if params.CancellationReason != "" {
		cancellationReason = pgtype.Text{String: params.CancellationReason, Valid: true}
	}

	_, err = s.repo.UpdateSubscriptionCancellation(ctx, repository.UpdateSubscriptionCancellationParams{
		ID:                 params.SubscriptionID,
		TenantID:           params.TenantID,
		CancelAtPeriodEnd:  params.CancelAtPeriodEnd,
		CancellationReason: cancellationReason,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription cancellation: %w", err)
	}

	// Create schedule event
	scheduleMetadata, _ := json.Marshal(map[string]string{
		"event":                "cancelled",
		"cancel_at_period_end": fmt.Sprintf("%t", params.CancelAtPeriodEnd),
		"reason":               params.CancellationReason,
	})

	_, err = s.repo.CreateSubscriptionScheduleEvent(ctx, repository.CreateSubscriptionScheduleEventParams{
		TenantID:       params.TenantID,
		SubscriptionID: params.SubscriptionID,
		EventType:      "cancel",
		Status:         "completed",
		ScheduledAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OrderID:        pgtype.UUID{},
		PaymentID:      pgtype.UUID{},
		Metadata:       scheduleMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule event: %w", err)
	}

	// Return updated subscription
	return s.GetSubscription(ctx, GetSubscriptionParams{
		TenantID:               params.TenantID,
		SubscriptionID:         params.SubscriptionID,
		IncludeUpcomingInvoice: false,
	})
}

// CreateCustomerPortalSession creates a Stripe Customer Portal session
func (s *subscriptionService) CreateCustomerPortalSession(ctx context.Context, params PortalSessionParams) (string, error) {
	// Get billing customer
	billingCustomer, err := s.repo.GetBillingCustomerForUser(ctx, repository.GetBillingCustomerForUserParams{
		UserID:   params.UserID,
		TenantID: params.TenantID,
		Provider: "stripe",
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoPaymentMethod
		}
		return "", fmt.Errorf("failed to get billing customer: %w", err)
	}

	// Create portal session
	session, err := s.billingProvider.CreateCustomerPortalSession(ctx, billing.CreatePortalSessionParams{
		CustomerID: billingCustomer.ProviderCustomerID,
		TenantID:   uuidToString(params.TenantID),
		ReturnURL:  params.ReturnURL,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}

	return session.URL, nil
}

// SyncSubscriptionFromWebhook updates local subscription from Stripe webhook
func (s *subscriptionService) SyncSubscriptionFromWebhook(ctx context.Context, params SyncSubscriptionParams) error {
	// Check if webhook event already processed (idempotency)
	_, err := s.repo.GetWebhookEventByProviderID(ctx, repository.GetWebhookEventByProviderIDParams{
		ProviderEventID: params.EventID,
		Provider:        "stripe",
		TenantID:        params.TenantID,
	})
	if err == nil {
		// Event already processed
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check webhook event: %w", err)
	}

	// Record webhook event for idempotency
	webhookPayload, _ := json.Marshal(map[string]string{
		"event_type":              params.EventType,
		"provider_subscription_id": params.ProviderSubscriptionID,
	})

	_, err = s.repo.CreateWebhookEvent(ctx, repository.CreateWebhookEventParams{
		TenantID:        params.TenantID,
		Provider:        "stripe",
		ProviderEventID: params.EventID,
		EventType:       params.EventType,
		Status:          "processing",
		Payload:         webhookPayload,
	})
	if err != nil {
		return fmt.Errorf("failed to create webhook event: %w", err)
	}

	// Get subscription from Stripe
	stripeSubscription, err := s.billingProvider.GetSubscription(ctx, billing.GetSubscriptionParams{
		SubscriptionID: params.ProviderSubscriptionID,
		TenantID:       uuidToString(params.TenantID),
	})
	if err != nil {
		return fmt.Errorf("failed to get Stripe subscription: %w", err)
	}

	// Get local subscription
	subscription, err := s.repo.GetSubscriptionByProviderID(ctx, repository.GetSubscriptionByProviderIDParams{
		ProviderSubscriptionID: pgtype.Text{String: params.ProviderSubscriptionID, Valid: true},
		Provider:               "stripe",
		TenantID:               params.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSubscriptionNotFound
		}
		return fmt.Errorf("failed to get local subscription: %w", err)
	}

	// Update local subscription status
	cancelledAt := pgtype.Timestamptz{Valid: false}
	if stripeSubscription.CanceledAt != nil {
		cancelledAt = pgtype.Timestamptz{Time: *stripeSubscription.CanceledAt, Valid: true}
	}

	_, err = s.repo.UpdateSubscriptionStatus(ctx, repository.UpdateSubscriptionStatusParams{
		ID:                 subscription.ID,
		TenantID:           params.TenantID,
		Status:             stripeSubscription.Status,
		CurrentPeriodStart: pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodStart, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodEnd, Valid: true},
		NextBillingDate:    pgtype.Timestamptz{Time: stripeSubscription.CurrentPeriodEnd, Valid: true},
		CancelAtPeriodEnd:  pgtype.Bool{Bool: stripeSubscription.CancelAtPeriodEnd, Valid: true},
		CancelledAt:        cancelledAt,
	})
	if err != nil {
		return fmt.Errorf("failed to update subscription status: %w", err)
	}

	return nil
}

// CreateOrderFromSubscriptionInvoice creates an order when subscription invoice is paid.
//
// Called from webhook handler when invoice.payment_succeeded event arrives with invoice.subscription set.
//
// Flow:
// 1. Get invoice details from Stripe
// 2. Get subscription from database by provider_subscription_id
// 3. Check idempotency via subscription_schedule (prevent duplicate orders for same invoice)
// 4. Create order with subscription_id link
// 5. Create order items from subscription items
// 6. Create payment record linking to Stripe invoice
// 7. Decrement inventory
// 8. Create subscription_schedule event for audit trail
func (s *subscriptionService) CreateOrderFromSubscriptionInvoice(ctx context.Context, invoiceID string, tenantID pgtype.UUID) (*OrderDetail, error) {
	// Step 1: Get invoice details from Stripe
	invoice, err := s.billingProvider.GetInvoice(ctx, billing.GetInvoiceParams{
		InvoiceID: invoiceID,
		TenantID:  uuidToString(tenantID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice from billing provider: %w", err)
	}

	// Validate it's a subscription invoice
	if invoice.SubscriptionID == "" {
		return nil, fmt.Errorf("invoice is not for a subscription")
	}

	// Step 2: Get local subscription by provider_subscription_id
	subscription, err := s.repo.GetSubscriptionByProviderID(ctx, repository.GetSubscriptionByProviderIDParams{
		TenantID:               tenantID,
		Provider:               "stripe",
		ProviderSubscriptionID: pgtype.Text{String: invoice.SubscriptionID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Step 3: Check if invoice already processed (idempotency)
	// Query uses: tenant_id, subscription_id, and metadata->>'invoice_id' = $3
	// The Metadata field is used as a string comparison against the JSON field
	existingEvent, err := s.repo.GetSubscriptionScheduleEventByInvoiceID(ctx, repository.GetSubscriptionScheduleEventByInvoiceIDParams{
		TenantID:       tenantID,
		SubscriptionID: subscription.ID,
		Metadata:       []byte(invoiceID), // Passed as string to compare against metadata->>'invoice_id'
	})
	if err == nil && existingEvent.ID.Valid {
		// Already processed - return the existing order
		return nil, ErrInvoiceAlreadyProcessed
	}
	// If error is sql.ErrNoRows, that's expected (invoice not yet processed)

	// Step 4: Get subscription items
	items, err := s.repo.ListSubscriptionItemsForSubscription(ctx, repository.ListSubscriptionItemsForSubscriptionParams{
		TenantID:       tenantID,
		SubscriptionID: subscription.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription items: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("subscription has no items")
	}

	// Step 5: Generate order number
	orderNumber, err := generateOrderNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate order number: %w", err)
	}

	// Step 6: Create order record
	// Use shipping address from subscription, billing address same as shipping for subscriptions
	order, err := s.repo.CreateOrder(ctx, repository.CreateOrderParams{
		TenantID:          tenantID,
		UserID:            subscription.UserID,
		OrderNumber:       orderNumber,
		OrderType:         "subscription",
		Status:            "confirmed", // Subscription payments are already successful
		SubscriptionID:    subscription.ID,
		SubtotalCents:     subscription.SubtotalCents,
		ShippingCents:     subscription.ShippingCents,
		TaxCents:          subscription.TaxCents,
		TotalCents:        subscription.TotalCents,
		Currency:          subscription.Currency,
		ShippingAddressID: subscription.ShippingAddressID,
		BillingAddressID:  subscription.ShippingAddressID, // Use shipping as billing for subscriptions
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Step 7: Create order items from subscription items
	orderItems := make([]repository.OrderItem, 0, len(items))
	for _, item := range items {
		// Build variant description
		variantDesc := ""
		if item.WeightValue.Valid {
			variantDesc = fmt.Sprintf("%s %s", item.WeightValue.Int.String(), item.WeightUnit)
		}
		if item.Grind != "" && item.Grind != "whole_bean" {
			if variantDesc != "" {
				variantDesc += " - "
			}
			variantDesc += item.Grind
		}

		orderItem, err := s.repo.CreateOrderItem(ctx, repository.CreateOrderItemParams{
			TenantID:           tenantID,
			OrderID:            order.ID,
			ProductSkuID:       item.ProductSkuID,
			ProductName:        item.ProductName,
			Sku:                item.Sku,
			VariantDescription: pgtype.Text{String: variantDesc, Valid: variantDesc != ""},
			Quantity:           item.Quantity,
			UnitPriceCents:     item.UnitPriceCents,
			TotalPriceCents:    item.UnitPriceCents * item.Quantity,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}
		orderItems = append(orderItems, orderItem)
	}

	// Step 8: Create payment record
	payment, err := s.repo.CreatePayment(ctx, repository.CreatePaymentParams{
		TenantID:          tenantID,
		BillingCustomerID: subscription.BillingCustomerID,
		Provider:          "stripe",
		ProviderPaymentID: invoice.PaymentIntentID,
		AmountCents:       int32(invoice.AmountPaidCents),
		Currency:          invoice.Currency,
		Status:            "succeeded",
		PaymentMethodID:   subscription.PaymentMethodID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Step 9: Decrement inventory for each item
	for _, item := range items {
		err := s.repo.DecrementSKUStock(ctx, repository.DecrementSKUStockParams{
			TenantID:          tenantID,
			ID:                item.ProductSkuID,
			InventoryQuantity: item.Quantity,
		})
		if err != nil {
			// Log warning but don't fail - inventory tracking is best-effort for MVP
			// In production, consider alerting on low stock
			log.Printf("WARNING: Failed to decrement stock for SKU %s: %v", item.Sku, err)
		}
	}

	// Step 10: Create subscription_schedule event for audit trail
	// Include invoice_id in metadata for idempotency lookup
	scheduleMetadata, _ := json.Marshal(map[string]string{
		"event":      "renewal",
		"invoice_id": invoiceID,
		"order_id":   uuidToString(order.ID),
	})
	_, err = s.repo.CreateSubscriptionScheduleEvent(ctx, repository.CreateSubscriptionScheduleEventParams{
		TenantID:       tenantID,
		SubscriptionID: subscription.ID,
		EventType:      "billing",
		Status:         "completed",
		ScheduledAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OrderID:        order.ID,
		PaymentID:      payment.ID,
		Metadata:       scheduleMetadata,
	})
	if err != nil {
		// Log warning but don't fail - schedule event is for audit trail
		log.Printf("WARNING: Failed to create subscription schedule event: %v", err)
	}

	// Step 11: Get shipping address for response
	shippingAddress, err := s.repo.GetAddressByID(ctx, subscription.ShippingAddressID)
	if err != nil {
		log.Printf("WARNING: Failed to get shipping address: %v", err)
		// Return order detail with empty addresses if address lookup fails
		return &OrderDetail{
			Order:   order,
			Items:   orderItems,
			Payment: payment,
		}, nil
	}

	// Step 12: Return order detail
	return &OrderDetail{
		Order:           order,
		Items:           orderItems,
		ShippingAddress: shippingAddress,
		BillingAddress:  shippingAddress, // Same as shipping for subscriptions
		Payment:         payment,
	}, nil
}

// Helper functions
// Note: uuidToString is defined in order.go

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "nil"
	}
	return t.Format(time.RFC3339)
}
