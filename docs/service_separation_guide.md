# Valkey vs NATS - Clear Separation of Concerns

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│   PostgreSQL    │    │     Valkey      │    │      NATS       │
│                 │    │                 │    │                 │
│  Primary Data   │    │ Cache & Session │    │ Event Streaming │
│   Long-term     │    │   Short-term    │    │   Workflows     │
│   ACID          │    │   Performance   │    │   Messaging     │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## **Valkey Responsibilities** 🔴

### **Primary Use Cases:**
- **Session Storage** - Shopping carts, user sessions, login tokens
- **Caching** - Product catalogs, pricing data, frequently accessed content
- **Rate Limiting** - API throttling, login attempt tracking
- **Temporary Data** - OTP codes, password reset tokens, temporary locks
- **Performance Optimization** - Reduce database load

### **Data Characteristics:**
- **Ephemeral** - Can be lost without major impact
- **High-frequency access** - Read/write heavy
- **Short-lived** - TTL-based expiration
- **Non-critical** - Performance optimization, not business logic

### **Implementation Examples:**

```go
// Shopping Cart in Valkey
type ValkeyCartService struct {
    client *valkey.Client
}

func (s *ValkeyCartService) GetCart(customerID int) (*Cart, error) {
    key := fmt.Sprintf("cart:%d", customerID)
    data, err := s.client.Get(ctx, key).Result()
    // Parse and return cart
}

func (s *ValkeyCartService) AddItem(customerID int, item CartItem) error {
    key := fmt.Sprintf("cart:%d", customerID)
    // Store cart with 24h TTL
    return s.client.SetEX(ctx, key, cartJSON, 24*time.Hour).Err()
}

// Product Cache in Valkey
func (s *ValkeyProductService) GetProduct(id int) (*Product, error) {
    key := fmt.Sprintf("product:%d", id)
    
    // Try cache first
    if cached, err := s.client.Get(ctx, key).Result(); err == nil {
        return parseProduct(cached), nil
    }
    
    // Fallback to database
    product, err := s.db.GetProduct(id)
    if err != nil {
        return nil, err
    }
    
    // Cache for 1 hour
    s.client.SetEX(ctx, key, productJSON, time.Hour)
    return product, nil
}

// Session Management in Valkey
func (s *ValkeySessionService) CreateSession(userID int) (string, error) {
    sessionID := generateSessionID()
    key := fmt.Sprintf("session:%s", sessionID)
    
    sessionData := map[string]interface{}{
        "user_id":    userID,
        "created_at": time.Now(),
        "last_seen":  time.Now(),
    }
    
    // 30-day session
    return sessionID, s.client.SetEX(ctx, key, sessionData, 30*24*time.Hour).Err()
}
```

---

## **NATS Responsibilities** 🟢

### **Primary Use Cases:**
- **Event Streaming** - Order processing workflows, audit trails
- **Business Events** - Order created, payment confirmed, item shipped
- **Service Communication** - Microservice messaging, async processing
- **Durable Messaging** - Events that must not be lost
- **Workflow Orchestration** - Multi-step business processes

### **Data Characteristics:**
- **Persistent** - Must not be lost (with JetStream)
- **Business-critical** - Core business logic events
- **Ordered** - Sequence matters for workflows
- **Auditable** - Compliance and debugging requirements

### **Implementation Examples:**

```go
// Order Processing Events in NATS
type NATSEventService struct {
    js nats.JetStreamContext
}

func (s *NATSEventService) PublishOrderCreated(order *Order) error {
    event := OrderCreatedEvent{
        OrderID:    order.ID,
        CustomerID: order.CustomerID,
        Total:      order.Total,
        Items:      order.Items,
        Timestamp:  time.Now(),
    }
    
    data, _ := json.Marshal(event)
    
    // Durable event - will be retried if processing fails
    _, err := s.js.Publish("orders.created", data)
    return err
}

// Event Handlers in NATS
func (s *OrderEventHandler) HandleOrderCreated(msg *nats.Msg) {
    var event OrderCreatedEvent
    json.Unmarshal(msg.Data, &event)
    
    // Business logic that MUST happen
    // 1. Update inventory
    // 2. Send confirmation email
    // 3. Create shipping label
    // 4. Update analytics
    
    if err := s.processOrder(event); err != nil {
        // NATS will retry this message
        return
    }
    
    // Acknowledge successful processing
    msg.Ack()
}

// Audit Trail in NATS
func (s *NATSAuditService) LogUserAction(userID int, action string, details map[string]interface{}) error {
    auditEvent := AuditEvent{
        UserID:    userID,
        Action:    action,
        Details:   details,
        Timestamp: time.Now(),
        TraceID:   getTraceID(),
    }
    
    data, _ := json.Marshal(auditEvent)
    
    // Persistent audit log
    _, err := s.js.Publish("audit.user_actions", data)
    return err
}
```

---

## **Configuration Files**

### **Valkey Configuration (`config/valkey/valkey.conf`)**
```conf
# Memory and Performance
maxmemory 512mb
maxmemory-policy allkeys-lru

# Persistence (for sessions)
save 900 1
save 300 10
save 60 10000

# Security
requirepass valkey_password_123
protected-mode yes

# Networking
tcp-keepalive 300
timeout 0

# Logging
loglevel notice
logfile ""
```

### **NATS Configuration (`config/nats/nats.conf`)**
```conf
# Server settings
server_name: "ecommerce-nats"
listen: 0.0.0.0:4222
http_port: 8222

# Authentication
authorization {
  token: "nats_token_123"
}

# JetStream (persistent messaging)
jetstream {
  store_dir: "/data"
  max_memory_store: 512MB
  max_file_store: 2GB
}

# Logging
log_file: "/dev/stdout"
debug: false
trace: false
```

---

## **When to Use Which Service**

### **Use Valkey For:**
✅ Shopping cart data  
✅ User session storage  
✅ Product catalog caching  
✅ API rate limiting  
✅ Temporary tokens (password reset, OTP)  
✅ Page/query result caching  
✅ Real-time data that can be regenerated  

### **Use NATS For:**
✅ Order processing workflows  
✅ Payment confirmation events  
✅ Inventory updates  
✅ Email/notification triggers  
✅ Audit logging  
✅ Service-to-service communication  
✅ Data synchronization between services  

### **Example: Order Processing Flow**

```go
// 1. User submits order (stored in PostgreSQL)
order := CreateOrder(orderData)

// 2. Clear cart from Valkey (temporary data)
valkeyService.ClearCart(customerID)

// 3. Publish order event to NATS (business workflow)
natsService.PublishOrderCreated(order)

// 4. NATS triggers multiple handlers:
//    - Update inventory (persistent)
//    - Send confirmation email
//    - Create shipping label
//    - Update analytics

// 5. Cache order details in Valkey for fast access
valkeyService.CacheOrder(order, 1*time.Hour)
```

---

## **Monitoring Separation**

### **Valkey Metrics to Monitor:**
- Memory usage and hit rates
- Cache performance and TTL effectiveness
- Session storage utilization
- Connection pool status

### **NATS Metrics to Monitor:**
- Message throughput and processing delays
- JetStream storage utilization
- Failed message delivery rates
- Consumer lag and acknowledgment rates

This separation ensures:
- **Valkey**: Fast, ephemeral data for performance
- **NATS**: Reliable, persistent events for business logic
- **Clear boundaries**: No overlap in responsibilities
- **Scalability**: Each system optimized for its use case