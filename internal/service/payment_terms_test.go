package service

import (
	"testing"
	"time"

	"github.com/dukerupert/freyja/internal/repository"
)

func TestCalculateDueDateFromTerms(t *testing.T) {
	svc := &paymentTermsService{}

	tests := []struct {
		name        string
		days        int32
		invoiceDate time.Time
		wantDate    time.Time
	}{
		{
			name:        "Net 30",
			days:        30,
			invoiceDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDate:    time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Net 15",
			days:        15,
			invoiceDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			wantDate:    time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Due on receipt (0 days)",
			days:        0,
			invoiceDate: time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC),
			wantDate:    time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC),
		},
		{
			name:        "Net 60 crossing month boundary",
			days:        60,
			invoiceDate: time.Date(2025, 11, 15, 0, 0, 0, 0, time.UTC),
			wantDate:    time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Net 7 week terms",
			days:        7,
			invoiceDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			wantDate:    time.Date(2025, 3, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Leap year February",
			days:        30,
			invoiceDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantDate:    time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := &repository.PaymentTerm{
				Days: tt.days,
			}

			got := svc.CalculateDueDateFromTerms(terms, tt.invoiceDate)

			if !got.Equal(tt.wantDate) {
				t.Errorf("CalculateDueDateFromTerms() = %v, want %v", got, tt.wantDate)
			}
		})
	}
}
