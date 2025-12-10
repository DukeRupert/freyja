package shipping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dukerupert/hiri/internal/shipping"
	"github.com/stretchr/testify/assert"
)

func TestFlatRateProvider_GetRates_SingleRate(t *testing.T) {
	rates := []shipping.FlatRate{
		{
			ServiceName: "Standard Shipping",
			ServiceCode: "STD",
			CostCents:   500,
			DaysMin:     3,
			DaysMax:     5,
		},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			Line1:      "123 Main St",
			City:       "Seattle",
			State:      "WA",
			PostalCode: "98101",
			Country:    "US",
		},
		Packages: []shipping.Package{
			{
				WeightGrams: 454, // 1 lb
				LengthCm:    20,
				WidthCm:     15,
				HeightCm:    10,
			},
		},
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.NoError(t, err)
	assert.Len(t, result, 1)

	rate := result[0]
	assert.Equal(t, "STD", rate.RateID)
	assert.Equal(t, "Flat Rate", rate.Carrier)
	assert.Equal(t, "Standard Shipping", rate.ServiceName)
	assert.Equal(t, "STD", rate.ServiceCode)
	assert.Equal(t, int64(500), rate.CostCents)
	assert.Equal(t, 3, rate.EstimatedDaysMin)
	assert.Equal(t, 5, rate.EstimatedDaysMax)
	assert.Nil(t, rate.ExpiresAt, "Flat rates should not expire")

	// Verify estimated delivery date is in the future
	assert.True(t, rate.EstimatedDeliveryDate.After(time.Now()))
}

func TestFlatRateProvider_GetRates_MultipleRates(t *testing.T) {
	rates := []shipping.FlatRate{
		{
			ServiceName: "Standard Shipping",
			ServiceCode: "STD",
			CostCents:   500,
			DaysMin:     3,
			DaysMax:     5,
		},
		{
			ServiceName: "Express Shipping",
			ServiceCode: "EXP",
			CostCents:   1500,
			DaysMin:     1,
			DaysMax:     2,
		},
		{
			ServiceName: "Priority Overnight",
			ServiceCode: "PRI",
			CostCents:   2500,
			DaysMin:     1,
			DaysMax:     1,
		},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			Line1:      "456 Oak Ave",
			City:       "Portland",
			State:      "OR",
			PostalCode: "97201",
			Country:    "US",
		},
		Packages: []shipping.Package{
			{WeightGrams: 340},
		},
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Verify all rates are returned with correct data
	for i, rate := range result {
		assert.Equal(t, rates[i].ServiceCode, rate.RateID)
		assert.Equal(t, "Flat Rate", rate.Carrier)
		assert.Equal(t, rates[i].ServiceName, rate.ServiceName)
		assert.Equal(t, rates[i].ServiceCode, rate.ServiceCode)
		assert.Equal(t, rates[i].CostCents, rate.CostCents)
		assert.Equal(t, rates[i].DaysMin, rate.EstimatedDaysMin)
		assert.Equal(t, rates[i].DaysMax, rate.EstimatedDaysMax)
	}
}

func TestFlatRateProvider_GetRates_EmptyRates(t *testing.T) {
	provider := shipping.NewFlatRateProvider([]shipping.FlatRate{})

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			Line1:      "789 Pine St",
			City:       "San Francisco",
			State:      "CA",
			PostalCode: "94102",
			Country:    "US",
		},
		Packages: []shipping.Package{
			{WeightGrams: 340},
		},
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.NoError(t, err)
	assert.Empty(t, result, "Should return empty slice when no rates configured")
}

func TestFlatRateProvider_GetRates_RequiresTenantID(t *testing.T) {
	rates := []shipping.FlatRate{
		{ServiceName: "Standard", ServiceCode: "STD", CostCents: 500, DaysMin: 3, DaysMax: 5},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		// TenantID is missing
		DestinationAddress: shipping.ShippingAddress{
			City:    "Denver",
			State:   "CO",
			Country: "US",
		},
		Packages: []shipping.Package{
			{WeightGrams: 340},
		},
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, shipping.ErrTenantRequired))
	assert.Nil(t, result)
}

func TestFlatRateProvider_GetRates_RequiresPackages(t *testing.T) {
	rates := []shipping.FlatRate{
		{ServiceName: "Standard", ServiceCode: "STD", CostCents: 500, DaysMin: 3, DaysMax: 5},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			City:    "Denver",
			State:   "CO",
			Country: "US",
		},
		Packages: []shipping.Package{}, // Empty packages
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, shipping.ErrNoPackages))
	assert.Nil(t, result)
}

func TestFlatRateProvider_GetRates_EstimatedDeliveryDate(t *testing.T) {
	rates := []shipping.FlatRate{
		{
			ServiceName: "Standard",
			ServiceCode: "STD",
			CostCents:   500,
			DaysMin:     3,
			DaysMax:     5,
		},
		{
			ServiceName: "Express",
			ServiceCode: "EXP",
			CostCents:   1500,
			DaysMin:     1,
			DaysMax:     2,
		},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			City:    "Denver",
			State:   "CO",
			Country: "US",
		},
		Packages: []shipping.Package{
			{WeightGrams: 340},
		},
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	now := time.Now()

	// Standard shipping: delivery in 5 days
	stdDelivery := result[0].EstimatedDeliveryDate
	expectedStd := now.AddDate(0, 0, 5)
	assert.True(t, stdDelivery.After(now))
	// Allow for some time variance (test execution time)
	assert.WithinDuration(t, expectedStd, stdDelivery, 5*time.Second)

	// Express shipping: delivery in 2 days
	expDelivery := result[1].EstimatedDeliveryDate
	expectedExp := now.AddDate(0, 0, 2)
	assert.True(t, expDelivery.After(now))
	assert.WithinDuration(t, expectedExp, expDelivery, 5*time.Second)
}

func TestFlatRateProvider_GetRates_IgnoresPackageDetails(t *testing.T) {
	rates := []shipping.FlatRate{
		{
			ServiceName: "Flat Rate",
			ServiceCode: "FLAT",
			CostCents:   1000,
			DaysMin:     2,
			DaysMax:     4,
		},
	}

	provider := shipping.NewFlatRateProvider(rates)

	// Try with different package sizes - should all return same rate
	packages := [][]shipping.Package{
		{{WeightGrams: 100, LengthCm: 10, WidthCm: 10, HeightCm: 10}},
		{{WeightGrams: 5000, LengthCm: 50, WidthCm: 50, HeightCm: 50}},
	}

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			City:    "Boston",
			State:   "MA",
			Country: "US",
		},
	}

	for _, pkgs := range packages {
		params.Packages = pkgs
		result, err := provider.GetRates(context.Background(), params)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(1000), result[0].CostCents, "Flat rate should ignore package details")
	}
}

func TestFlatRateProvider_GetRates_IgnoresDestination(t *testing.T) {
	rates := []shipping.FlatRate{
		{
			ServiceName: "Everywhere Flat",
			ServiceCode: "FLAT",
			CostCents:   750,
			DaysMin:     3,
			DaysMax:     7,
		},
	}

	provider := shipping.NewFlatRateProvider(rates)

	// Try different destinations - should all return same rate
	destinations := []shipping.ShippingAddress{
		{City: "New York", State: "NY", Country: "US"},
		{City: "Los Angeles", State: "CA", Country: "US"},
		{City: "Miami", State: "FL", Country: "US"},
		{City: "Anchorage", State: "AK", Country: "US"},
	}

	for _, dest := range destinations {
		params := shipping.RateParams{
			TenantID:           "tenant-123",
			DestinationAddress: dest,
			Packages:           []shipping.Package{{WeightGrams: 340}},
		}

		result, err := provider.GetRates(context.Background(), params)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(750), result[0].CostCents)
	}
}

func TestFlatRateProvider_GetRates_IgnoresServiceTypeFilter(t *testing.T) {
	rates := []shipping.FlatRate{
		{ServiceName: "Standard", ServiceCode: "STD", CostCents: 500, DaysMin: 3, DaysMax: 5},
		{ServiceName: "Express", ServiceCode: "EXP", CostCents: 1500, DaysMin: 1, DaysMax: 2},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			City:    "Chicago",
			State:   "IL",
			Country: "US",
		},
		Packages:     []shipping.Package{{WeightGrams: 340}},
		ServiceTypes: []string{"EXP"}, // Filter for express only
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.NoError(t, err)
	// FlatRateProvider ignores service type filter and returns all rates
	assert.Len(t, result, 2, "FlatRateProvider should return all configured rates regardless of filter")
}

func TestFlatRateProvider_CreateLabel_ReturnsNotImplemented(t *testing.T) {
	provider := shipping.NewFlatRateProvider([]shipping.FlatRate{})

	params := shipping.LabelParams{
		TenantID: "tenant-123",
		RateID:   "STD",
		DestinationAddress: shipping.ShippingAddress{
			Line1:   "123 Main St",
			City:    "Seattle",
			State:   "WA",
			Country: "US",
		},
	}

	label, err := provider.CreateLabel(context.Background(), params)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, shipping.ErrNotImplemented), "CreateLabel should return ErrNotImplemented")
	assert.Nil(t, label)
}

func TestFlatRateProvider_VoidLabel_ReturnsNotImplemented(t *testing.T) {
	provider := shipping.NewFlatRateProvider([]shipping.FlatRate{})

	params := shipping.VoidLabelParams{
		TenantID: "tenant-123",
		LabelID:  "label-123",
	}

	err := provider.VoidLabel(context.Background(), params)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, shipping.ErrNotImplemented), "VoidLabel should return ErrNotImplemented")
}

func TestFlatRateProvider_TrackShipment_ReturnsNotImplemented(t *testing.T) {
	provider := shipping.NewFlatRateProvider([]shipping.FlatRate{})

	tracking, err := provider.TrackShipment(context.Background(), "TRACK-123456")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, shipping.ErrNotImplemented), "TrackShipment should return ErrNotImplemented")
	assert.Nil(t, tracking)
}

func TestFlatRateProvider_ValidateAddress_ReturnsNotImplemented(t *testing.T) {
	provider := shipping.NewFlatRateProvider([]shipping.FlatRate{})

	params := shipping.ValidateAddressParams{
		TenantID: "tenant-123",
		Address: shipping.ShippingAddress{
			Line1:   "123 Main St",
			City:    "Seattle",
			State:   "WA",
			Country: "US",
		},
	}

	result, err := provider.ValidateAddress(context.Background(), params)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, shipping.ErrNotImplemented), "ValidateAddress should return ErrNotImplemented")
	assert.Nil(t, result)
}

func TestFlatRateProvider_NewConstructor(t *testing.T) {
	rates := []shipping.FlatRate{
		{ServiceName: "Test", ServiceCode: "TEST", CostCents: 100, DaysMin: 1, DaysMax: 3},
	}

	provider := shipping.NewFlatRateProvider(rates)

	assert.NotNil(t, provider, "NewFlatRateProvider should return non-nil provider")

	// Verify it implements the Provider interface
	var _ shipping.Provider = provider
}

func TestFlatRateProvider_GetRates_Idempotency(t *testing.T) {
	rates := []shipping.FlatRate{
		{ServiceName: "Standard", ServiceCode: "STD", CostCents: 500, DaysMin: 3, DaysMax: 5},
	}

	provider := shipping.NewFlatRateProvider(rates)

	params := shipping.RateParams{
		TenantID: "tenant-123",
		DestinationAddress: shipping.ShippingAddress{
			City:    "Nashville",
			State:   "TN",
			Country: "US",
		},
		Packages: []shipping.Package{{WeightGrams: 340}},
	}

	// Call multiple times with same params
	result1, err1 := provider.GetRates(context.Background(), params)
	result2, err2 := provider.GetRates(context.Background(), params)
	result3, err3 := provider.GetRates(context.Background(), params)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	// All results should have same structure (but delivery dates may vary slightly)
	assert.Equal(t, len(result1), len(result2))
	assert.Equal(t, len(result1), len(result3))
	assert.Equal(t, result1[0].CostCents, result2[0].CostCents)
	assert.Equal(t, result1[0].CostCents, result3[0].CostCents)
}

func TestFlatRateProvider_GetRates_RealWorldScenario(t *testing.T) {
	// Simulate real MVP configuration: Standard and Express shipping
	rates := []shipping.FlatRate{
		{
			ServiceName: "Standard Shipping",
			ServiceCode: "STD",
			CostCents:   500, // $5.00
			DaysMin:     3,
			DaysMax:     5,
		},
		{
			ServiceName: "Express Shipping",
			ServiceCode: "EXP",
			CostCents:   1500, // $15.00
			DaysMin:     1,
			DaysMax:     2,
		},
	}

	provider := shipping.NewFlatRateProvider(rates)

	// Customer ordering coffee from Seattle roaster
	params := shipping.RateParams{
		TenantID: "tenant-roaster-123",
		OriginAddress: shipping.ShippingAddress{
			Name:       "Freyja Coffee Roasters",
			Line1:      "100 Roaster Lane",
			City:       "Seattle",
			State:      "WA",
			PostalCode: "98101",
			Country:    "US",
		},
		DestinationAddress: shipping.ShippingAddress{
			Name:       "John Doe",
			Line1:      "456 Consumer St",
			City:       "Portland",
			State:      "OR",
			PostalCode: "97201",
			Country:    "US",
			Phone:      "555-1234",
			Email:      "john@example.com",
		},
		Packages: []shipping.Package{
			{
				WeightGrams: 340, // 12oz coffee bag
				LengthCm:    20,
				WidthCm:     15,
				HeightCm:    8,
			},
		},
	}

	result, err := provider.GetRates(context.Background(), params)

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify standard option
	std := result[0]
	assert.Equal(t, "Standard Shipping", std.ServiceName)
	assert.Equal(t, int64(500), std.CostCents)
	assert.Equal(t, 3, std.EstimatedDaysMin)
	assert.Equal(t, 5, std.EstimatedDaysMax)

	// Verify express option
	exp := result[1]
	assert.Equal(t, "Express Shipping", exp.ServiceName)
	assert.Equal(t, int64(1500), exp.CostCents)
	assert.Equal(t, 1, exp.EstimatedDaysMin)
	assert.Equal(t, 2, exp.EstimatedDaysMax)

	// Both should have valid delivery dates
	assert.True(t, std.EstimatedDeliveryDate.After(time.Now()))
	assert.True(t, exp.EstimatedDeliveryDate.After(time.Now()))
	assert.True(t, exp.EstimatedDeliveryDate.Before(std.EstimatedDeliveryDate),
		"Express delivery should be sooner than standard")
}
