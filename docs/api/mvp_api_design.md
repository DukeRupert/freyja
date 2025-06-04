# MVP E-commerce API Design

## Core Data Models

### Product
```go
type Product struct {
    ID          int       `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description" db:"description"`
    Price       int       `json:"price" db:"price"` // cents
    Stock       int       `json:"stock" db:"stock"`
    Active      bool      `json:"active" db:"active"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
```

### Customer
```go
type Customer struct {
    ID        int       `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    FirstName string    `json:"first_name" db:"first_name"`
    LastName  string    `json:"last_name" db:"last_name"`
    StripeID  string    `json:"-" db:"stripe_customer_id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

### Cart & CartItem
```go
type Cart struct {
    ID         int    `json:"id" db:"id"`
    CustomerID int    `json:"customer_id" db:"customer_id"`
    SessionID  string `json:"session_id" db:"session_id"` // for guest carts
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type CartItem struct {
    ID        int `json:"id" db:"id"`
    CartID    int `json:"cart_id" db:"cart_id"`
    ProductID int `json:"product_id" db:"product_id"`
    Quantity  int `json:"quantity" db:"quantity"`
    Price     int `json:"price" db:"price"` // locked-in price
}
```

### Order
```go
type Order struct {
    ID              int       `json:"id" db:"id"`
    CustomerID      int       `json:"customer_id" db:"customer_id"`
    Status          string    `json:"status" db:"status"`
    Total           int       `json:"total" db:"total"`
    StripeSessionID string    `json:"-" db:"stripe_session_id"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type OrderItem struct {
    ID        int    `json:"id" db:"id"`
    OrderID   int    `json:"order_id" db:"order_id"`
    ProductID int    `json:"product_id" db:"product_id"`
    Name      string `json:"name" db:"name"` // snapshot
    Quantity  int    `json:"quantity" db:"quantity"`
    Price     int    `json:"price" db:"price"`
}
```

---

## API Endpoints

### 1. Product Catalog

#### GET /api/v1/products
List all active products with current stock

**Response:**
```json
{
  "products": [
    {
      "id": 1,
      "name": "Ethiopian Yirgacheffe",
      "description": "Bright, floral notes with citrus finish",
      "price": 1800,
      "stock": 25,
      "active": true
    }
  ]
}
```

#### GET /api/v1/products/{id}
Get detailed product information

**Response:**
```json
{
  "id": 1,
  "name": "Ethiopian Yirgacheffe",
  "description": "Bright, floral notes with citrus finish",
  "price": 1800,
  "stock": 25,
  "active": true
}
```

---

### 2. Authentication (Simplified)

#### POST /api/v1/auth/register
Register new customer

**Request:**
```json
{
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "password": "secure_password"
}
```

**Response:**
```json
{
  "customer": {
    "id": 1,
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  },
  "token": "jwt_token_here"
}
```

#### POST /api/v1/auth/login
Login existing customer

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure_password"
}
```

**Response:**
```json
{
  "customer": { /* customer object */ },
  "token": "jwt_token_here"
}
```

---

### 3. Cart Management

#### GET /api/v1/cart
Get current cart (authenticated or by session)

**Headers:** `Authorization: Bearer {token}` OR `X-Session-ID: {session_id}`

**Response:**
```json
{
  "cart": {
    "id": 1,
    "items": [
      {
        "id": 1,
        "product_id": 1,
        "name": "Ethiopian Yirgacheffe",
        "quantity": 2,
        "price": 1800
      }
    ],
    "total": 3600
  }
}
```

#### POST /api/v1/cart/items
Add item to cart

**Request:**
```json
{
  "product_id": 1,
  "quantity": 2
}
```

**Response:**
```json
{
  "cart_item": {
    "id": 1,
    "product_id": 1,
    "quantity": 2,
    "price": 1800
  }
}
```

#### PUT /api/v1/cart/items/{id}
Update cart item quantity

**Request:**
```json
{
  "quantity": 3
}
```

#### DELETE /api/v1/cart/items/{id}
Remove item from cart

**Response:** `204 No Content`

---

### 4. Checkout Process

#### POST /api/v1/checkout
Create Stripe checkout session

**Request:**
```json
{
  "success_url": "https://yoursite.com/success",
  "cancel_url": "https://yoursite.com/cart"
}
```

**Response:**
```json
{
  "checkout_session_id": "cs_stripe_session_id",
  "checkout_url": "https://checkout.stripe.com/pay/cs_..."
}
```

#### POST /api/v1/webhooks/stripe
Handle Stripe webhooks (payment confirmations)

**Headers:** `Stripe-Signature: webhook_signature`

**Request:** Raw Stripe webhook payload

**Response:** `200 OK` with `{"received": true}`

---

### 5. Order Management

#### GET /api/v1/orders
Get customer's order history

**Response:**
```json
{
  "orders": [
    {
      "id": 1,
      "status": "completed",
      "total": 3600,
      "created_at": "2025-06-03T10:00:00Z",
      "items": [
        {
          "name": "Ethiopian Yirgacheffe",
          "quantity": 2,
          "price": 1800
        }
      ]
    }
  ]
}
```

#### GET /api/v1/orders/{id}
Get specific order details

**Response:**
```json
{
  "id": 1,
  "status": "processing",
  "total": 3600,
  "created_at": "2025-06-03T10:00:00Z",
  "items": [
    {
      "name": "Ethiopian Yirgacheffe",
      "quantity": 2,
      "price": 1800
    }
  ],
  "tracking_number": "1Z999AA1234567890"
}
```

---

### 6. Admin Endpoints (Basic)

#### GET /api/v1/admin/orders
List all orders with filters

**Query Parameters:**
- `status` - filter by order status
- `limit` - pagination limit
- `offset` - pagination offset

#### PUT /api/v1/admin/orders/{id}/status
Update order status

**Request:**
```json
{
  "status": "shipped",
  "tracking_number": "1Z999AA1234567890"
}
```

#### PUT /api/v1/admin/products/{id}/stock
Update product stock

**Request:**
```json
{
  "stock": 50
}
```

---

## Database Schema (PostgreSQL)

### Core Tables
```sql
-- Products
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price INTEGER NOT NULL, -- cents
    stock INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Customers
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    password_hash VARCHAR(255) NOT NULL,
    stripe_customer_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Carts
CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id),
    session_id VARCHAR(255), -- for guest carts
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT cart_owner_check CHECK (
        (customer_id IS NOT NULL) OR (session_id IS NOT NULL)
    )
);

-- Cart Items
CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER REFERENCES carts(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price INTEGER NOT NULL, -- locked price
    created_at TIMESTAMP DEFAULT NOW()
);

-- Orders
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total INTEGER NOT NULL,
    stripe_session_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Order Items
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    name VARCHAR(255) NOT NULL, -- snapshot
    quantity INTEGER NOT NULL,
    price INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Events (for event sourcing)
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Essential Indexes
```sql
CREATE INDEX idx_products_active ON products(active) WHERE active = true;
CREATE INDEX idx_carts_customer ON carts(customer_id);
CREATE INDEX idx_carts_session ON carts(session_id);
CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_events_type_aggregate ON events(event_type, aggregate_id);
```

---

## Go Handler Structure Example

```go
// Product handlers
func (h *ProductHandler) GetProducts(c echo.Context) error
func (h *ProductHandler) GetProduct(c echo.Context) error

// Auth handlers  
func (h *AuthHandler) Register(c echo.Context) error
func (h *AuthHandler) Login(c echo.Context) error

// Cart handlers
func (h *CartHandler) GetCart(c echo.Context) error
func (h *CartHandler) AddItem(c echo.Context) error
func (h *CartHandler) UpdateItem(c echo.Context) error
func (h *CartHandler) RemoveItem(c echo.Context) error

// Checkout handlers
func (h *CheckoutHandler) CreateSession(c echo.Context) error
func (h *WebhookHandler) HandleStripe(c echo.Context) error

// Order handlers
func (h *OrderHandler) GetOrders(c echo.Context) error
func (h *OrderHandler) GetOrder(c echo.Context) error

// Admin handlers
func (h *AdminHandler) GetOrders(c echo.Context) error
func (h *AdminHandler) UpdateOrderStatus(c echo.Context) error
func (h *AdminHandler) UpdateStock(c echo.Context) error
```

This MVP API gives you everything needed to:
1. Display products to customers
2. Manage shopping carts (guest and authenticated)
3. Process payments through Stripe
4. Create and track orders
5. Basic admin functions for fulfillment

The design prioritizes simplicity while maintaining the event-driven architecture foundation for future expansion.