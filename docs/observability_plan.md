# E-commerce Observability Plan with Prometheus & Grafana

## Architecture Overview

```
Application --> Prometheus Metrics --> Prometheus Server --> Grafana Dashboards
     |              |                        |                      |
     |              |                        |                      |
     v              v                        v                      v
Event Stream --> Log Aggregation --> Alert Manager --> Notification Channels
     |
     v
Audit Log Database
```

## Core Metrics Categories

### 1. Business Metrics (Revenue & Sales)
- **Revenue tracking** - real-time sales, conversion rates
- **Product performance** - top sellers, inventory turnover
- **Customer behavior** - cart abandonment, repeat purchases
- **Order fulfillment** - processing times, shipping delays

### 2. Technical Metrics (Performance & Health)
- **API performance** - response times, error rates, throughput
- **Database performance** - query times, connection pools
- **Cache performance** - hit/miss rates, memory usage
- **Third-party integrations** - Stripe/shipping provider latencies

### 3. Infrastructure Metrics (System Health)
- **Application health** - CPU, memory, disk usage
- **Network metrics** - request volumes, bandwidth
- **Error tracking** - panic rates, failed requests
- **Event processing** - NATS queue depths, processing delays

---

## Prometheus Metrics Implementation

### Custom Metrics in Go Application

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Business Metrics
    OrdersTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ecommerce_orders_total",
            Help: "Total number of orders by status",
        },
        []string{"status", "payment_method"},
    )
    
    RevenueTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ecommerce_revenue_cents_total",
            Help: "Total revenue in cents",
        },
        []string{"product_category", "customer_type"},
    )
    
    CartAbandonmentRate = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ecommerce_cart_abandonment_rate",
            Help: "Cart abandonment rate percentage",
        },
        []string{"time_period"},
    )
    
    ProductStock = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ecommerce_product_stock",
            Help: "Current product stock levels",
        },
        []string{"product_id", "product_name"},
    )
    
    // Technical Metrics
    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    DatabaseQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "database_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
        },
        []string{"query_type", "table"},
    )
    
    ExternalAPIRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "external_api_request_duration_seconds",
            Help:    "External API request duration",
            Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
        },
        []string{"provider", "operation"},
    )
    
    EventProcessingDelay = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "event_processing_delay_seconds",
            Help:    "Delay between event creation and processing",
            Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
        },
        []string{"event_type"},
    )
    
    CacheOperations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_operations_total",
            Help: "Total cache operations",
        },
        []string{"operation", "result"}, // get/set/delete, hit/miss/error
    )
)
```

### Middleware Implementation

```go
// HTTP Request Metrics Middleware
func PrometheusMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()
            
            err := next(c)
            
            duration := time.Since(start).Seconds()
            statusCode := strconv.Itoa(c.Response().Status)
            
            HTTPRequestDuration.WithLabelValues(
                c.Request().Method,
                c.Path(),
                statusCode,
            ).Observe(duration)
            
            return err
        }
    }
}

// Database Metrics Wrapper
func (r *PostgresProductRepository) GetByID(ctx context.Context, id int) (*Product, error) {
    start := time.Now()
    defer func() {
        DatabaseQueryDuration.WithLabelValues("select", "products").Observe(
            time.Since(start).Seconds(),
        )
    }()
    
    return r.queries.GetProduct(ctx, id)
}

// Event Processing Metrics
func (h *OrderEventHandler) HandleOrderCreated(ctx context.Context, event Event) error {
    // Calculate processing delay
    delay := time.Since(event.Timestamp).Seconds()
    EventProcessingDelay.WithLabelValues("order.created").Observe(delay)
    
    // Record business metrics
    OrdersTotal.WithLabelValues("created", "stripe").Inc()
    
    return h.processOrder(ctx, event)
}
```

---

## Structured Logging for Audit Trails

### Log Structure

```go
package logging

import (
    "context"
    "time"
    
    "github.com/sirupsen/logrus"
)

type AuditEvent struct {
    EventID     string                 `json:"event_id"`
    Timestamp   time.Time              `json:"timestamp"`
    UserID      *int                   `json:"user_id,omitempty"`
    SessionID   *string                `json:"session_id,omitempty"`
    Action      string                 `json:"action"`
    Resource    string                 `json:"resource"`
    ResourceID  *string                `json:"resource_id,omitempty"`
    Changes     map[string]interface{} `json:"changes,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    IPAddress   string                 `json:"ip_address"`
    UserAgent   string                 `json:"user_agent"`
    Success     bool                   `json:"success"`
    ErrorMsg    *string                `json:"error_message,omitempty"`
}

func LogAuditEvent(ctx context.Context, event AuditEvent) {
    logrus.WithFields(logrus.Fields{
        "audit":        true,
        "event_id":     event.EventID,
        "user_id":      event.UserID,
        "action":       event.Action,
        "resource":     event.Resource,
        "resource_id":  event.ResourceID,
        "success":      event.Success,
        "ip_address":   event.IPAddress,
        "changes":      event.Changes,
        "metadata":     event.Metadata,
    }).Info("Audit Event")
}

// Usage Examples
func (h *OrderHandler) UpdateOrderStatus(c echo.Context) error {
    orderID := c.Param("id")
    var req UpdateOrderRequest
    
    // ... validation logic ...
    
    oldOrder, _ := h.services.Orders.GetByID(c.Request().Context(), orderID)
    err := h.services.Orders.UpdateStatus(c.Request().Context(), orderID, req.Status)
    
    LogAuditEvent(c.Request().Context(), AuditEvent{
        EventID:    generateUUID(),
        Timestamp:  time.Now(),
        UserID:     getUserID(c),
        Action:     "order.status_updated",
        Resource:   "order",
        ResourceID: &orderID,
        Changes: map[string]interface{}{
            "old_status": oldOrder.Status,
            "new_status": req.Status,
        },
        IPAddress:  c.RealIP(),
        UserAgent:  c.Request().UserAgent(),
        Success:    err == nil,
        ErrorMsg:   errorToString(err),
    })
    
    return err
}
```

---

## Grafana Dashboard Design

### Dashboard Categories

#### 1. Executive Dashboard
- **Real-time revenue** - current day, week, month
- **Order volume trends** - hourly, daily patterns
- **Top performing products** - revenue and quantity
- **Customer acquisition** - new vs returning customers
- **Geographic distribution** - orders by location

#### 2. Operations Dashboard
- **Order fulfillment pipeline** - orders by status
- **Inventory alerts** - low stock warnings
- **Processing times** - order to shipment duration
- **Error rates** - failed payments, shipping issues
- **Customer service metrics** - support ticket volume

#### 3. Technical Dashboard
- **API Performance** - response times, error rates
- **Database Performance** - query times, connection pool usage
- **Cache Performance** - hit rates, memory usage
- **External Dependencies** - Stripe, shipping provider health
- **Infrastructure Health** - CPU, memory, disk usage

#### 4. Customer Experience Dashboard
- **Conversion funnel** - product view → cart → checkout → order
- **Cart abandonment** - rates and recovery metrics
- **Payment success rates** - by provider and method
- **Customer journey timing** - time to complete purchase
- **User experience metrics** - page load times, errors

### Sample Grafana Queries

```promql
# Revenue per hour
rate(ecommerce_revenue_cents_total[1h]) * 3600 / 100

# Order conversion rate
(
  rate(ecommerce_orders_total{status="completed"}[5m]) /
  rate(ecommerce_orders_total{status="created"}[5m])
) * 100

# Average order processing time
histogram_quantile(0.95, 
  rate(order_processing_duration_seconds_bucket[5m])
)

# Top products by revenue
topk(10, 
  rate(ecommerce_revenue_cents_total[1h]) * 3600
)

# API error rate
(
  rate(http_request_duration_seconds_count{status_code=~"5.."}[5m]) /
  rate(http_request_duration_seconds_count[5m])
) * 100

# Database query performance
histogram_quantile(0.95, 
  rate(database_query_duration_seconds_bucket[5m])
)

# Cache hit rate
(
  rate(cache_operations_total{result="hit"}[5m]) /
  rate(cache_operations_total{operation="get"}[5m])
) * 100

# Event processing lag
histogram_quantile(0.95, 
  rate(event_processing_delay_seconds_bucket[5m])
)
```

---

## Alerting Strategy

### Critical Business Alerts

```yaml
# alerts.yml
groups:
- name: business_critical
  rules:
  - alert: RevenueDropped
    expr: rate(ecommerce_revenue_cents_total[1h]) < 1000 # $10/hour
    for: 30m
    labels:
      severity: critical
    annotations:
      summary: "Revenue has dropped significantly"
      
  - alert: HighCartAbandonmentRate
    expr: ecommerce_cart_abandonment_rate > 80
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "Cart abandonment rate is above 80%"
      
  - alert: ProductOutOfStock
    expr: ecommerce_product_stock < 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Product {{ $labels.product_name }} is low on stock"

- name: technical_critical
  rules:
  - alert: HighErrorRate
    expr: rate(http_request_duration_seconds_count{status_code=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High error rate detected"
      
  - alert: SlowDatabaseQueries
    expr: histogram_quantile(0.95, rate(database_query_duration_seconds_bucket[5m])) > 1
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Database queries are running slowly"
      
  - alert: ExternalAPIDown
    expr: up{job="stripe"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "External API is down"
```

### Notification Channels

```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@yourcoffee.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'
  - match:
      severity: warning
    receiver: 'warning-alerts'

receivers:
- name: 'web.hook'
  webhook_configs:
  - url: 'http://127.0.0.1:5001/'

- name: 'critical-alerts'
  email_configs:
  - to: 'on-call@yourcoffee.com'
    subject: 'CRITICAL: {{ .GroupLabels.alertname }}'
  slack_configs:
  - api_url: 'YOUR_SLACK_WEBHOOK'
    channel: '#alerts'
    title: 'Critical Alert'

- name: 'warning-alerts'
  email_configs:
  - to: 'team@yourcoffee.com'
    subject: 'WARNING: {{ .GroupLabels.alertname }}'
```

---

## Implementation Timeline

### Phase 1: Foundation (Week 1-2)
- Set up Prometheus server and basic Go metrics
- Implement HTTP request metrics middleware
- Create basic technical dashboard in Grafana
- Set up basic alerting for system health

### Phase 2: Business Metrics (Week 3-4)
- Add order and revenue tracking metrics
- Implement audit logging for all critical actions
- Create executive dashboard showing revenue trends
- Add inventory and product performance metrics

### Phase 3: Advanced Observability (Week 5-6)
- Add customer journey tracking
- Implement detailed event processing metrics
- Create operations dashboard for fulfillment team
- Set up business-critical alerts

### Phase 4: Optimization (Week 7-8)
- Add performance profiling and detailed tracing
- Implement predictive alerts (trending issues)
- Create customer experience dashboard
- Add A/B testing metrics infrastructure

---

## Security and Compliance Considerations

### Audit Log Requirements
- **Immutable storage** - audit logs cannot be modified
- **Data retention** - comply with regulations (7 years for financial)
- **Access logging** - who accessed what data when
- **PCI compliance** - secure handling of payment data
- **GDPR compliance** - customer data access and deletion tracking

### Sensitive Data Handling
- **No PII in metrics** - use hashed customer IDs
- **Secure log transport** - encrypted log shipping
- **Access control** - role-based dashboard access
- **Data anonymization** - mask sensitive data in non-prod

This observability strategy gives you comprehensive visibility into your coffee business operations while maintaining security and compliance requirements.