# Service Abstractions

This document describes the interface-based abstractions for external services in Freyja.

## Philosophy

All external dependencies (storage, email, billing, shipping) are defined as interfaces with concrete implementations. This allows:

- **Easy testing**: Mock implementations for unit tests
- **Swappable providers**: Change from local storage → S3, or SMTP → Postmark without changing business logic
- **Clear boundaries**: Service layer depends on interfaces, not concrete implementations
- **Gradual migration**: Start simple implementations (local storage, SMTP), upgrade to production services later

## Storage

**Interface**: `storage.Storage`

**Implementations**:
- `LocalStorage` (MVP): Stores files on local filesystem, serves via HTTP
- `S3Storage` (Post-MVP): Stores files in AWS S3 or S3-compatible storage

**Usage**:
```go
// Initialize based on config
var fileStorage storage.Storage
if config.Storage == "s3" {
    fileStorage, _ = storage.NewS3Storage(config.S3Bucket, config.S3Region)
} else {
    fileStorage, _ = storage.NewLocalStorage("./web/static/uploads", "/uploads")
}

// Services depend on the interface
productService := service.NewProductService(repo, fileStorage, tenantID)

// Upload product image
url, err := fileStorage.Put(ctx, "products/uuid/image.jpg", file, "image/jpeg")
```

**Configuration**:
```env
# Local storage (MVP)
STORAGE_TYPE=local
STORAGE_PATH=./web/static/uploads
STORAGE_URL=/uploads

# S3 storage (Post-MVP)
STORAGE_TYPE=s3
S3_BUCKET=freyja-uploads
S3_REGION=us-west-2
```

## Email

**Interface**: `email.Sender`

**Implementations**:
- `SMTPSender` (MVP): Simple SMTP for Mailhog development
- `SMTPTLSSender`: SMTP with TLS for production SMTP relays
- `PostmarkSender` (Post-MVP): Postmark API
- `ResendSender` (Post-MVP): Resend API
- `SESSender` (Post-MVP): AWS SES

**Usage**:
```go
// Initialize based on config
var emailSender email.Sender
if config.EmailProvider == "postmark" {
    emailSender = email.NewPostmarkSender(config.PostmarkAPIKey)
} else {
    emailSender = email.NewSMTPSender(
        config.SMTPHost,
        config.SMTPPort,
        config.SMTPUsername,
        config.SMTPPassword,
        config.SMTPFrom,
    )
}

// Services depend on the interface
orderService := service.NewOrderService(repo, emailSender, tenantID)

// Send email
err := emailSender.Send(ctx, &email.Email{
    To:      []string{"customer@example.com"},
    Subject: "Order Confirmation",
    HTMLBody: renderedTemplate,
})
```

**Configuration**:
```env
# SMTP (MVP)
EMAIL_PROVIDER=smtp
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=noreply@freyja.local

# Postmark (Post-MVP)
EMAIL_PROVIDER=postmark
POSTMARK_API_KEY=your-api-key
POSTMARK_FROM=orders@yourdomain.com
```

## Billing

**Interface**: `billing.Provider`

**Implementations**:
- `StripeProvider`: Stripe payment processing and invoicing

**Core Payment Methods**:
- `CreateCustomer` / `UpdateCustomer` / `GetCustomer`: Customer management
- `CreatePaymentIntent`: One-time payment processing
- `CreateSubscription` / `UpdateSubscription` / `CancelSubscription`: Recurring billing

**Invoice Methods** (for wholesale/B2B):
- `CreateInvoice`: Create draft invoice for a customer
- `AddInvoiceItem`: Add line items to invoice
- `FinalizeInvoice`: Finalize draft invoice
- `SendInvoice`: Send invoice to customer via Stripe
- `VoidInvoice`: Cancel/void an open invoice
- `PayInvoice`: Mark invoice as paid (for manual payments)
- `GetInvoice`: Retrieve invoice details from Stripe

**Usage**:
```go
// Initialize
billingProvider := billing.NewStripeProvider(
    config.StripeSecretKey,
    config.StripeWebhookSecret,
)

// Services depend on the interface
checkoutService := service.NewCheckoutService(repo, billingProvider, tenantID)
invoiceService := service.NewInvoiceService(repo, paymentTerms, billingProvider, tenantID)

// Create payment intent (B2C)
intent, err := billingProvider.CreatePaymentIntent(ctx, billing.PaymentIntentParams{
    Amount:   totalCents,
    Currency: "usd",
    CustomerID: stripeCustomerID,
})

// Create invoice (B2B wholesale)
invoice, err := billingProvider.CreateInvoice(ctx, billing.CreateInvoiceParams{
    CustomerID:     stripeCustomerID,
    TenantID:       tenantID,
    DaysUntilDue:   30,
    CollectionMethod: "send_invoice",
    Description:    "Wholesale order #12345",
})
```

## Dependency Injection Pattern

**Main application setup**:

```go
func run() error {
    // Load config
    cfg, err := internal.NewConfig()
    if err != nil {
        return err
    }

    // Initialize database
    db, err := sql.Open("pgx", cfg.DatabaseUrl)
    if err != nil {
        return err
    }
    defer db.Close()

    // Create repository
    repo := repository.New(db)

    // Initialize external service implementations
    fileStorage, err := storage.NewLocalStorage(
        cfg.StoragePath,
        cfg.StorageURL,
    )
    if err != nil {
        return err
    }

    emailSender := email.NewSMTPSender(
        cfg.Email.Host,
        cfg.Email.Port,
        cfg.Email.Username,
        cfg.Email.Password,
        cfg.Email.From,
    )

    billingProvider := billing.NewStripeProvider(
        cfg.Stripe.SecretKey,
        cfg.Stripe.WebhookSecret,
    )

    // Parse tenant ID
    tenantID, _ := uuid.Parse(cfg.TenantID)

    // Initialize services with dependencies
    productService := service.NewProductService(repo, tenantID)
    cartService := service.NewCartService(repo, tenantID)
    // orderService := service.NewOrderService(repo, emailSender, tenantID)
    // checkoutService := service.NewCheckoutService(repo, billingProvider, tenantID)

    // Initialize handlers with services
    productHandler := handler.NewProductListHandler(productService, templates)
    cartHandler := handler.NewCartHandler(cartService, templates)

    // Setup routes
    // ...
}
```

## Testing with Mocks

**Example test using mock storage**:

```go
type mockStorage struct {
    storage.Storage
    putCalled bool
    putKey    string
}

func (m *mockStorage) Put(ctx context.Context, key string, content io.Reader, contentType string) (string, error) {
    m.putCalled = true
    m.putKey = key
    return "/uploads/" + key, nil
}

func TestProductService_UploadImage(t *testing.T) {
    mock := &mockStorage{}
    service := NewProductService(repo, mock, tenantID)

    url, err := service.UploadProductImage(ctx, productID, imageFile)

    assert.NoError(t, err)
    assert.True(t, mock.putCalled)
    assert.Contains(t, mock.putKey, "products/")
}
```

## Benefits

1. **No vendor lock-in**: Switch from Mailhog to Postmark by changing configuration
2. **Testable**: Mock interfaces in unit tests without external dependencies
3. **Gradual upgrades**: Start simple (local storage), upgrade to production (S3) when needed
4. **Clear contracts**: Interface documents exactly what the service needs
5. **Compile-time safety**: Type checker ensures implementations match interfaces
6. **Single responsibility**: Each implementation focused on one provider

## Future Implementations

### Shipping
- Interface: `shipping.Provider`
- Implementations: `EasyPostProvider`, `ShippoProvider`, `ShipStationProvider`, `ManualProvider`

### Analytics
- Interface: `analytics.Tracker`
- Implementations: `SegmentTracker`, `GoogleAnalyticsTracker`, `PlausibleTracker`

### Search
- Interface: `search.Index`
- Implementations: `PostgresSearch` (MVP), `MeiliSearchIndex`, `AlgoliaIndex`
