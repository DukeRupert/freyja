Step 3: Application Logic
When displaying products to a user:

Check if user is logged in and get their user_id
Query products filtered by:

Standard products (based on their price list visibility)
White-label products where white_label_customer_id = user_id


Never show products where visibility = 'hidden' unless they're white-label for that specific customer

Step 4: Admin UI Enhancements
Product creation flow:

Standard product: Normal flow
White-label product:

Select base product (dropdown of existing products)
Select customer (dropdown of wholesale-approved users)
Enter white-label name/description
Upload white-label images
Create matching SKUs (can pre-populate from base product)