# Shipping Setup

Configure how shipping rates are calculated and labels are created.

## Shipping Providers

### Flat Rate
Simple fixed-price shipping:
- Standard: $7.95 (5-7 business days)
- Express: $14.95 (2-3 business days)

Best for:
- Simple pricing
- Consistent shipping costs
- Getting started quickly

### EasyPost
Real-time carrier integration:
- USPS, UPS, FedEx, DHL rates
- Discounted label purchasing
- Automatic tracking
- Address validation

Best for:
- Accurate real-time rates
- Multiple carrier options
- Integrated label printing

## Setting Up Shipping

1. Go to **Settings > Shipping**
2. Select your shipping provider
3. Configure provider settings
4. Save changes

## Provider: Flat Rate

### Configuration

| Setting | Description |
|---------|-------------|
| Standard Rate | Price for standard shipping |
| Standard Days | Delivery estimate (e.g., "5-7 business days") |
| Express Rate | Price for express shipping |
| Express Days | Delivery estimate (e.g., "2-3 business days") |

### Using Flat Rate
- Rates shown at checkout
- Customer chooses standard or express
- You purchase labels separately (through carrier or shipping service)

## Provider: EasyPost

### Requirements
- EasyPost account ([easypost.com](https://easypost.com))
- API key from EasyPost dashboard
- Carrier accounts configured in EasyPost

### Configuration

| Setting | Description |
|---------|-------------|
| API Key | Your EasyPost API key |
| Ship From Address | Your return address |

### Setting Up EasyPost

1. Create EasyPost account
2. Get API key from dashboard
3. Enable carrier accounts (USPS, UPS, etc.)
4. Enter API key in Freyja
5. Configure ship-from address

### Features
- **Real-time rates**: Actual carrier rates at checkout
- **Label purchasing**: Buy labels directly in Freyja
- **Address validation**: Verify addresses before shipping
- **Tracking**: Automatic tracking updates

## Ship-From Address

Your ship-from address is used for:
- Calculating shipping rates
- Return address on labels
- Carrier pickup location

Ensure it's accurate and complete.

## Free Shipping

To offer free shipping:

### Flat Rate
Set rates to $0.00

### EasyPost
Not directly supported—you'd absorb the shipping cost

### Conditional Free Shipping
(e.g., "Free shipping over $50")
Currently not built-in. Contact support for options.

## Testing Shipping

After configuration:
1. Add items to cart
2. Proceed to checkout
3. Enter a shipping address
4. Verify rates appear correctly
5. Complete test order
6. Create test label (void after testing)

## Troubleshooting

### No Rates Shown
- Check provider configuration
- Verify API key (EasyPost)
- Check ship-from address

### Wrong Rates
- Verify package dimensions
- Check carrier account settings
- Review address accuracy

### Label Issues
- Verify address is valid
- Check carrier service availability
- Try alternative carrier

---

Previous: [Tax Configuration](tax.md) | Next: [Email Settings](email.md)
