-- +goose Up
-- +goose StatementBegin

-- Seed default legal pages for existing tenants
-- Note: This uses UPSERT to avoid duplicates if run multiple times

INSERT INTO tenant_pages (tenant_id, slug, title, content, meta_description, last_updated_label, is_published)
SELECT
    t.id,
    'privacy',
    'Privacy Policy',
    '<p>At ' || t.name || ', we are committed to protecting your privacy. This Privacy Policy explains how we collect, use, disclose, and safeguard your information when you visit our website or make a purchase.</p>

<h2>Information We Collect</h2>

<h3>Personal Information</h3>
<p>When you place an order or create an account, we collect information such as:</p>
<ul>
<li>Name and contact information (email address, phone number)</li>
<li>Billing and shipping addresses</li>
<li>Payment information (processed securely through Stripe)</li>
<li>Order history and preferences</li>
</ul>

<h3>Automatically Collected Information</h3>
<p>When you visit our website, we may automatically collect certain information, including:</p>
<ul>
<li>IP address and browser type</li>
<li>Device information</li>
<li>Pages visited and time spent on our site</li>
<li>Referring website addresses</li>
</ul>

<h2>How We Use Your Information</h2>
<p>We use the information we collect to:</p>
<ul>
<li>Process and fulfill your orders</li>
<li>Manage your account and subscriptions</li>
<li>Send order confirmations and shipping updates</li>
<li>Respond to your inquiries and provide customer support</li>
<li>Send promotional communications (with your consent)</li>
<li>Improve our website and services</li>
<li>Prevent fraud and enhance security</li>
</ul>

<h2>Payment Security</h2>
<p>All payment information is processed securely through Stripe. We do not store your complete credit card information on our servers.</p>

<h2>Contact Us</h2>
<p>If you have any questions about this Privacy Policy, please contact us at:</p>
<p><strong>Email:</strong> ' || t.email || '</p>',
    'Our privacy policy explains how we collect, use, and protect your personal information.',
    'December 2024',
    true
FROM tenants t
ON CONFLICT (tenant_id, slug) DO NOTHING;

INSERT INTO tenant_pages (tenant_id, slug, title, content, meta_description, last_updated_label, is_published)
SELECT
    t.id,
    'terms',
    'Terms of Service',
    '<p>Welcome to ' || t.name || '. By accessing or using our website and services, you agree to be bound by these Terms of Service.</p>

<h2>1. Acceptance of Terms</h2>
<p>By accessing our website, creating an account, or making a purchase, you acknowledge that you have read, understood, and agree to be bound by these Terms of Service and our Privacy Policy.</p>

<h2>2. Account Registration</h2>
<p>To access certain features, you may need to create an account. You agree to provide accurate information and maintain the security of your credentials.</p>

<h2>3. Products and Pricing</h2>
<p>We strive to provide accurate product descriptions and pricing. We reserve the right to modify prices and cancel orders in case of errors.</p>

<h2>4. Subscriptions</h2>
<p>Subscriptions auto-renew. You may pause, modify, or cancel at any time through your account dashboard.</p>

<h2>5. Limitation of Liability</h2>
<p>To the fullest extent permitted by law, ' || t.name || ' shall not be liable for any indirect, incidental, special, or consequential damages.</p>

<h2>Contact Us</h2>
<p>Questions? Contact us at:</p>
<p><strong>Email:</strong> ' || t.email || '</p>',
    'Terms and conditions for using our website and services.',
    'December 2024',
    true
FROM tenants t
ON CONFLICT (tenant_id, slug) DO NOTHING;

INSERT INTO tenant_pages (tenant_id, slug, title, content, meta_description, last_updated_label, is_published)
SELECT
    t.id,
    'shipping',
    'Shipping & Returns',
    '<h2>Shipping Information</h2>

<h3>Processing Time</h3>
<p>We roast to order to ensure maximum freshness. Orders are typically processed and shipped within 1-3 business days.</p>

<h3>Shipping Methods</h3>
<p>We offer Standard (5-7 days), Priority (2-3 days), and Express (1-2 days) shipping. Rates calculated at checkout.</p>

<h3>Free Shipping</h3>
<p>Free standard shipping on qualifying orders. Subscribers enjoy free shipping on all subscription orders.</p>

<h2>Returns & Refunds</h2>

<h3>Satisfaction Guarantee</h3>
<p>We stand behind our coffee quality. Contact us within 14 days if unsatisfied.</p>

<h3>Damaged Products</h3>
<p>Contact us within 7 days of delivery for damaged items. We''ll replace or refund.</p>

<h2>Contact Us</h2>
<p><strong>Email:</strong> ' || t.email || '</p>
<p><em>Include your order number in all correspondence.</em></p>',
    'Information about shipping methods, delivery times, and return policy.',
    'December 2024',
    true
FROM tenants t
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Only delete pages created by this migration (default slugs)
DELETE FROM tenant_pages WHERE slug IN ('privacy', 'terms', 'shipping');
-- +goose StatementEnd
