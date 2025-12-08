package page

import "fmt"

// Default page content in HTML format (compatible with Tiptap output)

func defaultPrivacyContent(storeName, contactEmail string) string {
	return fmt.Sprintf(`<p>At %s, we are committed to protecting your privacy. This Privacy Policy explains how we collect, use, disclose, and safeguard your information when you visit our website or make a purchase.</p>

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

<h2>Information Sharing</h2>
<p>We do not sell, trade, or rent your personal information to third parties. We may share your information with:</p>
<ul>
<li><strong>Service providers:</strong> Companies that help us operate our business (payment processors, shipping carriers, email services)</li>
<li><strong>Legal requirements:</strong> When required by law or to protect our rights</li>
<li><strong>Business transfers:</strong> In connection with a merger, acquisition, or sale of assets</li>
</ul>

<h2>Payment Security</h2>
<p>All payment information is processed securely through Stripe. We do not store your complete credit card information on our servers. Stripe is PCI-DSS compliant, ensuring the highest standards of payment security.</p>

<h2>Cookies</h2>
<p>We use cookies to enhance your browsing experience, remember your preferences, and understand how you interact with our site. You can control cookie settings through your browser preferences.</p>

<h2>Your Rights</h2>
<p>You have the right to:</p>
<ul>
<li>Access the personal information we hold about you</li>
<li>Request correction of inaccurate information</li>
<li>Request deletion of your personal information</li>
<li>Opt out of marketing communications</li>
<li>Lodge a complaint with a supervisory authority</li>
</ul>

<h2>Data Retention</h2>
<p>We retain your personal information for as long as necessary to fulfill the purposes outlined in this policy, unless a longer retention period is required by law.</p>

<h2>Children's Privacy</h2>
<p>Our website is not intended for children under 13 years of age. We do not knowingly collect personal information from children.</p>

<h2>Changes to This Policy</h2>
<p>We may update this Privacy Policy from time to time. We will notify you of any changes by posting the new policy on this page and updating the "Last updated" date.</p>

<h2>Contact Us</h2>
<p>If you have any questions about this Privacy Policy, please contact us at:</p>
<p><strong>Email:</strong> %s</p>`, storeName, contactEmail)
}

func defaultTermsContent(storeName, contactEmail string) string {
	return fmt.Sprintf(`<p>Welcome to %s. By accessing or using our website and services, you agree to be bound by these Terms of Service. Please read them carefully.</p>

<h2>1. Acceptance of Terms</h2>
<p>By accessing our website, creating an account, or making a purchase, you acknowledge that you have read, understood, and agree to be bound by these Terms of Service and our Privacy Policy.</p>

<h2>2. Account Registration</h2>
<p>To access certain features of our website, you may need to create an account. You agree to:</p>
<ul>
<li>Provide accurate and complete information</li>
<li>Maintain the security of your account credentials</li>
<li>Notify us immediately of any unauthorized use</li>
<li>Accept responsibility for all activities under your account</li>
</ul>

<h2>3. Products and Pricing</h2>
<p>We strive to provide accurate product descriptions and pricing. However:</p>
<ul>
<li>Product images are for illustration and may vary slightly from actual products</li>
<li>We reserve the right to modify prices without prior notice</li>
<li>In the event of pricing errors, we reserve the right to cancel orders</li>
<li>Availability of products is subject to change</li>
</ul>

<h2>4. Orders and Payment</h2>
<p>When you place an order:</p>
<ul>
<li>You agree to pay all charges at the prices in effect when incurred</li>
<li>You authorize us to charge your payment method for the total amount</li>
<li>We reserve the right to refuse or cancel any order for any reason</li>
<li>Orders are subject to availability and confirmation</li>
</ul>

<h2>5. Subscriptions</h2>
<p>If you subscribe to our coffee delivery service:</p>
<ul>
<li>Your subscription will automatically renew at the selected frequency</li>
<li>You will be charged automatically using your saved payment method</li>
<li>You may pause, modify, or cancel your subscription at any time through your account</li>
<li>Subscription prices may change with advance notice</li>
<li>Cancellations take effect after the current billing period</li>
</ul>

<h2>6. Wholesale Accounts</h2>
<p>Wholesale accounts are subject to additional terms:</p>
<ul>
<li>Wholesale pricing is available only to approved business accounts</li>
<li>We reserve the right to approve or deny wholesale applications</li>
<li>Payment terms (e.g., Net 15, Net 30) are subject to credit approval</li>
<li>Wholesale accounts may be subject to minimum order requirements</li>
<li>We reserve the right to revoke wholesale status at any time</li>
</ul>

<h2>7. Intellectual Property</h2>
<p>All content on our website, including text, graphics, logos, images, and software, is the property of %s or its content suppliers and is protected by intellectual property laws. You may not reproduce, distribute, or create derivative works without our express permission.</p>

<h2>8. Prohibited Uses</h2>
<p>You agree not to:</p>
<ul>
<li>Use our website for any unlawful purpose</li>
<li>Attempt to gain unauthorized access to any portion of the website</li>
<li>Interfere with or disrupt the website's functionality</li>
<li>Upload malicious code or content</li>
<li>Collect user information without consent</li>
<li>Impersonate any person or entity</li>
</ul>

<h2>9. Limitation of Liability</h2>
<p>To the fullest extent permitted by law, %s shall not be liable for any indirect, incidental, special, consequential, or punitive damages arising from your use of our website or products. Our total liability shall not exceed the amount you paid for the specific product or service giving rise to the claim.</p>

<h2>10. Disclaimer of Warranties</h2>
<p>Our website and products are provided "as is" without warranties of any kind, either express or implied. We do not warrant that our website will be uninterrupted, error-free, or free of viruses or other harmful components.</p>

<h2>11. Indemnification</h2>
<p>You agree to indemnify and hold harmless %s, its officers, directors, employees, and agents from any claims, damages, or expenses arising from your use of our website or violation of these Terms.</p>

<h2>12. Governing Law</h2>
<p>These Terms shall be governed by and construed in accordance with the laws of the state in which %s is registered, without regard to conflicts of law principles.</p>

<h2>13. Changes to Terms</h2>
<p>We reserve the right to modify these Terms at any time. Changes will be effective immediately upon posting. Your continued use of our website after changes constitutes acceptance of the modified Terms.</p>

<h2>14. Contact Information</h2>
<p>If you have any questions about these Terms of Service, please contact us at:</p>
<p><strong>Email:</strong> %s</p>`, storeName, storeName, storeName, storeName, storeName, contactEmail)
}

func defaultShippingContent(storeName, contactEmail string) string {
	return fmt.Sprintf(`<h2>Shipping Information</h2>

<h3>Processing Time</h3>
<p>We roast to order to ensure maximum freshness. Orders are typically processed and shipped within 1-3 business days. During peak seasons, processing may take slightly longer.</p>

<h3>Shipping Methods & Rates</h3>
<p>We offer the following shipping options:</p>
<table>
<thead>
<tr>
<th>Method</th>
<th>Delivery Time</th>
<th>Cost</th>
</tr>
</thead>
<tbody>
<tr>
<td>Standard Shipping</td>
<td>5-7 business days</td>
<td>Calculated at checkout</td>
</tr>
<tr>
<td>Priority Shipping</td>
<td>2-3 business days</td>
<td>Calculated at checkout</td>
</tr>
<tr>
<td>Express Shipping</td>
<td>1-2 business days</td>
<td>Calculated at checkout</td>
</tr>
</tbody>
</table>

<h3>Free Shipping</h3>
<p>We offer free standard shipping on orders over a certain threshold. Active subscribers enjoy free shipping on all subscription orders.</p>

<h3>Shipping Destinations</h3>
<p>We currently ship to all 50 U.S. states. International shipping is not available at this time.</p>

<h3>Order Tracking</h3>
<p>Once your order ships, you'll receive a confirmation email with tracking information. You can also track your order status in your account dashboard.</p>

<h2>Subscription Shipping</h2>
<p>Subscription orders are processed and shipped according to your selected delivery frequency. You'll receive a shipping notification for each delivery. To change your shipping address or delivery schedule, visit your account settings or contact us before your next billing date.</p>

<h2>Returns & Refunds</h2>

<h3>Our Satisfaction Guarantee</h3>
<p>We stand behind the quality of our coffee. If you're not completely satisfied with your purchase, we'll make it right.</p>

<h3>Damaged or Defective Products</h3>
<p>If your order arrives damaged or defective, please contact us within 7 days of delivery. We'll send a replacement at no additional cost or issue a full refund.</p>

<h3>Quality Issues</h3>
<p>Coffee is a perishable product, and taste is subjective. If you're unsatisfied with the quality or taste of your coffee, please contact us within 14 days of delivery. We'll work with you to find a suitable solution, which may include a replacement or store credit.</p>

<h3>Order Errors</h3>
<p>If you received the wrong product or your order was incomplete, contact us immediately. We'll ship the correct items at no extra charge.</p>

<h3>Non-Returnable Items</h3>
<p>Due to the perishable nature of coffee, we cannot accept returns of opened products unless there is a quality or defect issue. Sale items and gift cards are final sale.</p>

<h2>Order Cancellations</h2>

<h3>One-Time Orders</h3>
<p>Orders can be cancelled within 1 hour of placement, before they enter our roasting queue. Once an order is being processed, it cannot be cancelled. Please contact us immediately if you need to cancel an order.</p>

<h3>Subscription Cancellations</h3>
<p>You can pause, skip, or cancel your subscription at any time through your account dashboard. Cancellations take effect after the current billing period ends. Refunds are not provided for the current billing cycle.</p>

<h2>Wholesale Orders</h2>
<p>Wholesale orders may have different shipping terms and return policies. Please refer to your wholesale agreement or contact our wholesale team for specific information.</p>

<h2>Contact Us</h2>
<p>Have questions about shipping or need to initiate a return? We're here to help:</p>
<p><strong>Email:</strong> %s</p>
<p><em>Please include your order number in all correspondence.</em></p>`, contactEmail)
}
