package shipping

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/EasyPost/easypost-go/v5"
)

// Conversion constants for metric to imperial units.
// Precision chosen for shipping accuracy (< 0.01% error).
const (
	cmToInchRatio  = 0.393701 // 1 cm = 0.393701 inches
	gramsToOzRatio = 0.035274 // 1 gram = 0.035274 ounces
	rateExpiration = 24 * time.Hour
)

// EasyPostProvider implements the Provider interface using EasyPost API.
type EasyPostProvider struct {
	client *easypost.Client
	logger *slog.Logger
}

// EasyPostConfig contains configuration for the EasyPost provider.
type EasyPostConfig struct {
	APIKey string
	Logger *slog.Logger // Optional: defaults to slog.Default()
}

// NewEasyPostProvider creates a new EasyPost shipping provider.
func NewEasyPostProvider(cfg EasyPostConfig) (*EasyPostProvider, error) {
	if cfg.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	client := easypost.New(cfg.APIKey)

	return &EasyPostProvider{
		client: client,
		logger: logger,
	}, nil
}

// GetRates returns available shipping options for a shipment.
// MVP: Only single-package shipments are supported.
func (p *EasyPostProvider) GetRates(ctx context.Context, params RateParams) ([]Rate, error) {
	// Validate required fields
	if params.TenantID == "" {
		return nil, ErrTenantRequired
	}
	if params.OriginAddress.Line1 == "" {
		return nil, ErrOriginRequired
	}
	if len(params.Packages) == 0 {
		return nil, ErrNoPackages
	}
	if len(params.Packages) > 1 {
		return nil, ErrMultiPackageNotSupported
	}

	logger := p.logger.With(
		"tenant_id", params.TenantID,
		"destination_country", params.DestinationAddress.Country,
		"destination_state", params.DestinationAddress.State,
	)
	logger.Info("fetching shipping rates")

	// Build addresses
	fromAddress := p.toEasyPostAddress(params.OriginAddress)
	toAddress := p.toEasyPostAddress(params.DestinationAddress)
	parcel := p.toEasyPostParcel(params.Packages[0])

	// Create shipment with tenant_id in reference for later validation
	shipment, err := p.client.CreateShipment(
		&easypost.Shipment{
			FromAddress: fromAddress,
			ToAddress:   toAddress,
			Parcel:      parcel,
			Reference:   params.TenantID, // Store tenant_id for security validation
		},
	)
	if err != nil {
		logger.Error("failed to create shipment", "error", err)
		return nil, fmt.Errorf("failed to get rates: %w", err)
	}

	if len(shipment.Rates) == 0 {
		logger.Warn("no rates available for shipment")
		return nil, ErrNoRates
	}

	// Calculate rate expiration time
	var shipmentCreatedAt time.Time
	if shipment.CreatedAt != nil {
		shipmentCreatedAt = shipment.CreatedAt.AsTime()
	} else {
		shipmentCreatedAt = time.Now()
	}
	expiresAt := shipmentCreatedAt.Add(rateExpiration)

	// Convert EasyPost rates to our Rate type
	rates := make([]Rate, 0, len(shipment.Rates))
	for _, r := range shipment.Rates {
		rate, err := p.fromEasyPostRate(r, shipment.ID, &expiresAt)
		if err != nil {
			logger.Warn("failed to parse rate", "carrier", r.Carrier, "error", err)
			continue
		}
		rates = append(rates, rate)
	}

	// Filter by service types if specified
	if len(params.ServiceTypes) > 0 {
		rates = p.filterRatesByService(rates, params.ServiceTypes)
	}

	logger.Info("rates fetched successfully",
		"rate_count", len(rates),
		"shipment_id", shipment.ID,
	)

	return rates, nil
}

// CreateLabel generates a shipping label.
// Includes idempotency check - if shipment already purchased, returns existing label.
func (p *EasyPostProvider) CreateLabel(ctx context.Context, params LabelParams) (*Label, error) {
	// Validate required fields
	if params.TenantID == "" {
		return nil, ErrTenantRequired
	}

	logger := p.logger.With(
		"tenant_id", params.TenantID,
		"rate_id", params.RateID,
	)
	logger.Info("creating shipping label")

	// Parse compound rate ID
	shipmentID, rateID, err := parseRateID(params.RateID)
	if err != nil {
		return nil, ErrInvalidRate
	}

	// Get the shipment
	shipment, err := p.client.GetShipment(shipmentID)
	if err != nil {
		logger.Error("failed to get shipment", "error", err)
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	// SECURITY: Validate tenant ownership
	if shipment.Reference != params.TenantID {
		logger.Warn("tenant mismatch detected",
			"expected", params.TenantID,
			"actual", shipment.Reference,
		)
		return nil, ErrTenantMismatch
	}

	// IDEMPOTENCY: Check if already purchased
	if shipment.PostageLabel != nil && shipment.PostageLabel.LabelURL != "" {
		logger.Info("returning existing label (idempotent)")
		createdAt := time.Now()
		if shipment.CreatedAt != nil {
			createdAt = shipment.CreatedAt.AsTime()
		}
		return &Label{
			LabelID:        shipment.ID,
			TrackingNumber: shipment.TrackingCode,
			LabelURL:       shipment.PostageLabel.LabelURL,
			CreatedAt:      createdAt,
		}, nil
	}

	// Find the selected rate
	var selectedRate *easypost.Rate
	for _, r := range shipment.Rates {
		if r.ID == rateID {
			selectedRate = r
			break
		}
	}

	if selectedRate == nil {
		return nil, ErrInvalidRate
	}

	// Buy the shipment with the selected rate
	boughtShipment, err := p.client.BuyShipment(shipmentID, selectedRate, "")
	if err != nil {
		logger.Error("failed to purchase label", "error", err)
		return nil, fmt.Errorf("failed to purchase label: %w", err)
	}

	createdAt := time.Now()
	if boughtShipment.CreatedAt != nil {
		createdAt = boughtShipment.CreatedAt.AsTime()
	}

	logger.Info("label purchased successfully",
		"tracking_number", boughtShipment.TrackingCode,
		"label_id", boughtShipment.ID,
	)

	return &Label{
		LabelID:        boughtShipment.ID,
		TrackingNumber: boughtShipment.TrackingCode,
		LabelURL:       boughtShipment.PostageLabel.LabelURL,
		CreatedAt:      createdAt,
	}, nil
}

// VoidLabel cancels a shipping label and requests a refund.
func (p *EasyPostProvider) VoidLabel(ctx context.Context, params VoidLabelParams) error {
	if params.TenantID == "" {
		return ErrTenantRequired
	}

	logger := p.logger.With(
		"tenant_id", params.TenantID,
		"label_id", params.LabelID,
	)
	logger.Info("voiding shipping label")

	// Get shipment to validate tenant ownership
	shipment, err := p.client.GetShipment(params.LabelID)
	if err != nil {
		logger.Error("failed to get shipment", "error", err)
		return fmt.Errorf("failed to get shipment: %w", err)
	}

	// SECURITY: Validate tenant ownership
	if shipment.Reference != params.TenantID {
		logger.Warn("tenant mismatch detected")
		return ErrTenantMismatch
	}

	_, err = p.client.RefundShipment(params.LabelID)
	if err != nil {
		logger.Error("failed to void label", "error", err)
		return fmt.Errorf("failed to void label: %w", err)
	}

	logger.Info("label voided successfully")
	return nil
}

// TrackShipment gets tracking information for a shipment.
func (p *EasyPostProvider) TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	logger := p.logger.With("tracking_number", trackingNumber)
	logger.Info("fetching tracking info")

	tracker, err := p.client.CreateTracker(
		&easypost.CreateTrackerOptions{
			TrackingCode: trackingNumber,
		},
	)
	if err != nil {
		logger.Error("failed to create tracker", "error", err)
		return nil, fmt.Errorf("failed to create tracker: %w", err)
	}

	logger.Info("tracking info fetched", "status", tracker.Status)
	return p.fromEasyPostTracker(tracker), nil
}

// ValidateAddress validates and potentially corrects a shipping address.
func (p *EasyPostProvider) ValidateAddress(ctx context.Context, params ValidateAddressParams) (*AddressValidation, error) {
	if params.TenantID == "" {
		return nil, ErrTenantRequired
	}

	logger := p.logger.With(
		"tenant_id", params.TenantID,
		"city", params.Address.City,
		"state", params.Address.State,
	)
	logger.Info("validating address")

	epAddress := p.toEasyPostAddress(params.Address)

	verified, err := p.client.CreateAddress(epAddress, &easypost.CreateAddressOptions{
		Verify: true,
	})
	if err != nil {
		logger.Error("failed to validate address", "error", err)
		return nil, fmt.Errorf("failed to validate address: %w", err)
	}

	result := &AddressValidation{
		Status:          AddressValid,
		OriginalAddress: params.Address,
	}

	// Check if address was modified
	suggestedAddr := p.fromEasyPostAddress(verified)
	if !addressesEqual(params.Address, suggestedAddr) {
		result.Status = AddressValidWithChanges
		result.SuggestedAddress = &suggestedAddr
	}

	// Check for verification errors
	if verified.Verifications != nil && verified.Verifications.Delivery != nil {
		if !verified.Verifications.Delivery.Success {
			result.Status = AddressInvalid
			for _, e := range verified.Verifications.Delivery.Errors {
				result.Messages = append(result.Messages, e.Message)
			}
		}
	}

	logger.Info("address validated", "status", result.Status)
	return result, nil
}

// toEasyPostAddress converts our ShippingAddress to EasyPost Address.
func (p *EasyPostProvider) toEasyPostAddress(addr ShippingAddress) *easypost.Address {
	return &easypost.Address{
		Name:    addr.Name,
		Company: addr.Company,
		Street1: addr.Line1,
		Street2: addr.Line2,
		City:    addr.City,
		State:   addr.State,
		Zip:     addr.PostalCode,
		Country: addr.Country,
		Phone:   addr.Phone,
		Email:   addr.Email,
	}
}

// fromEasyPostAddress converts EasyPost Address to our ShippingAddress.
func (p *EasyPostProvider) fromEasyPostAddress(addr *easypost.Address) ShippingAddress {
	return ShippingAddress{
		Name:       addr.Name,
		Company:    addr.Company,
		Line1:      addr.Street1,
		Line2:      addr.Street2,
		City:       addr.City,
		State:      addr.State,
		PostalCode: addr.Zip,
		Country:    addr.Country,
		Phone:      addr.Phone,
		Email:      addr.Email,
	}
}

// toEasyPostParcel converts our Package to EasyPost Parcel.
func (p *EasyPostProvider) toEasyPostParcel(pkg Package) *easypost.Parcel {
	return &easypost.Parcel{
		// EasyPost uses inches for dimensions and ounces for weight
		Length: cmToInches(pkg.LengthCm),
		Width:  cmToInches(pkg.WidthCm),
		Height: cmToInches(pkg.HeightCm),
		Weight: gramsToOunces(pkg.WeightGrams),
	}
}

// fromEasyPostRate converts EasyPost Rate to our Rate type.
func (p *EasyPostProvider) fromEasyPostRate(r *easypost.Rate, shipmentID string, expiresAt *time.Time) (Rate, error) {
	// Parse delivery days
	daysMin := 1
	daysMax := 5
	if r.DeliveryDays > 0 {
		daysMin = r.DeliveryDays
		daysMax = r.DeliveryDays
	}

	// Parse estimated delivery date
	var estDelivery time.Time
	if r.DeliveryDate != nil {
		estDelivery = r.DeliveryDate.AsTime()
	}
	if estDelivery.IsZero() {
		estDelivery = time.Now().AddDate(0, 0, daysMax)
	}

	// Convert rate to cents
	costCents, err := dollarsToCents(r.Rate)
	if err != nil {
		return Rate{}, fmt.Errorf("failed to parse rate amount: %w", err)
	}

	return Rate{
		// Encode shipment ID with rate ID so we can buy later
		RateID:                fmt.Sprintf("%s:%s", shipmentID, r.ID),
		Carrier:               r.Carrier,
		ServiceName:           r.Service,
		ServiceCode:           r.Service,
		CostCents:             costCents,
		EstimatedDaysMin:      daysMin,
		EstimatedDaysMax:      daysMax,
		EstimatedDeliveryDate: estDelivery,
		ExpiresAt:             expiresAt,
	}, nil
}

// fromEasyPostTracker converts EasyPost Tracker to our TrackingInfo.
func (p *EasyPostProvider) fromEasyPostTracker(t *easypost.Tracker) *TrackingInfo {
	info := &TrackingInfo{
		TrackingNumber: t.TrackingCode,
		Status:         t.Status,
	}

	// Parse estimated delivery
	if t.EstDeliveryDate != nil {
		info.EstimatedDeliveryDate = t.EstDeliveryDate.AsTime()
	}

	// Convert tracking details to events
	for _, detail := range t.TrackingDetails {
		event := TrackingEvent{
			Status:      detail.Status,
			Description: detail.Message,
		}
		if detail.DateTime != "" {
			if dt, err := time.Parse(time.RFC3339, detail.DateTime); err == nil {
				event.Timestamp = dt
			}
		}
		if detail.TrackingLocation != nil {
			event.Location = fmt.Sprintf("%s, %s %s",
				detail.TrackingLocation.City,
				detail.TrackingLocation.State,
				detail.TrackingLocation.Zip,
			)
		}
		info.Events = append(info.Events, event)
	}

	return info
}

// filterRatesByService filters rates to only include specified service types.
func (p *EasyPostProvider) filterRatesByService(rates []Rate, services []string) []Rate {
	serviceSet := make(map[string]bool)
	for _, s := range services {
		serviceSet[s] = true
	}

	var filtered []Rate
	for _, r := range rates {
		if serviceSet[r.ServiceCode] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// parseRateID splits a compound rate ID into shipment ID and rate ID.
func parseRateID(rateID string) (shipmentID, epRateID string, err error) {
	shipmentID, epRateID, ok := strings.Cut(rateID, ":")
	if !ok || shipmentID == "" || epRateID == "" {
		return "", "", ErrInvalidRateIDFormat
	}
	return shipmentID, epRateID, nil
}

// addressesEqual compares two addresses for equality.
func addressesEqual(a, b ShippingAddress) bool {
	return a.Name == b.Name &&
		a.Company == b.Company &&
		a.Line1 == b.Line1 &&
		a.Line2 == b.Line2 &&
		a.City == b.City &&
		a.State == b.State &&
		a.PostalCode == b.PostalCode &&
		a.Country == b.Country
}

// Unit conversion helpers

func cmToInches(cm int32) float64 {
	return float64(cm) * cmToInchRatio
}

func gramsToOunces(grams int32) float64 {
	return float64(grams) * gramsToOzRatio
}

// dollarsToCents converts a dollar amount string to cents.
// Handles formats like "5.25", "5", "5.1", "5.05".
func dollarsToCents(dollars string) (int64, error) {
	dollars = strings.TrimSpace(dollars)
	if dollars == "" {
		return 0, ErrInvalidAmount("", nil)
	}

	amount, err := strconv.ParseFloat(dollars, 64)
	if err != nil {
		return 0, ErrInvalidAmount(dollars, err)
	}

	// Convert to cents, rounding to nearest cent
	cents := int64(math.Round(amount * 100))
	return cents, nil
}
