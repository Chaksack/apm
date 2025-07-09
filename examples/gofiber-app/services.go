package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrDatabaseDown      = errors.New("database is down")
	ErrCacheDown         = errors.New("cache is down")
	ErrExternalAPIDown   = errors.New("external API is down")
)

// Services holds all application services
type Services struct {
	db        *DatabaseService
	cache     *CacheService
	user      *UserService
	product   *ProductService
	order     *OrderService
	analytics *AnalyticsService
	external  *ExternalAPIService
}

// NewServices creates and initializes all services
func NewServices(cfg *AppConfig) (*Services, error) {
	// Initialize database service
	db := NewDatabaseService(cfg.DBEnabled)

	// Initialize cache service
	cache := NewCacheService(cfg.CacheEnabled)

	// Initialize external API service with circuit breaker
	external := NewExternalAPIService()

	// Initialize business services
	services := &Services{
		db:        db,
		cache:     cache,
		user:      NewUserService(db, cache),
		product:   NewProductService(db, cache),
		order:     NewOrderService(db, cache),
		analytics: NewAnalyticsService(db),
		external:  external,
	}

	return services, nil
}

// Close closes all services
func (s *Services) Close() {
	// Clean up resources if needed
}

// DatabaseService simulates database operations
type DatabaseService struct {
	enabled bool
	mu      sync.RWMutex
	data    map[string]interface{}
}

func NewDatabaseService(enabled bool) *DatabaseService {
	return &DatabaseService{
		enabled: enabled,
		data:    make(map[string]interface{}),
	}
}

func (db *DatabaseService) HealthCheck(ctx context.Context) bool {
	_, span := tracer.Start(ctx, "db.health-check")
	defer span.End()

	// Simulate random health status
	healthy := db.enabled && rand.Float32() > 0.1

	span.SetAttributes(attribute.Bool("healthy", healthy))

	return healthy
}

func (db *DatabaseService) Get(ctx context.Context, key string) (interface{}, error) {
	_, span := tracer.Start(ctx, "db.get")
	defer span.End()

	span.SetAttributes(attribute.String("key", key))

	if !db.enabled {
		span.RecordError(ErrDatabaseDown)
		span.SetStatus(codes.Error, "Database is disabled")
		return nil, ErrDatabaseDown
	}

	// Simulate database latency
	time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)

	db.mu.RLock()
	value, exists := db.data[key]
	db.mu.RUnlock()

	if !exists {
		span.SetStatus(codes.Error, "Key not found")
		return nil, ErrNotFound
	}

	return value, nil
}

func (db *DatabaseService) Set(ctx context.Context, key string, value interface{}) error {
	_, span := tracer.Start(ctx, "db.set")
	defer span.End()

	span.SetAttributes(attribute.String("key", key))

	if !db.enabled {
		span.RecordError(ErrDatabaseDown)
		span.SetStatus(codes.Error, "Database is disabled")
		return ErrDatabaseDown
	}

	// Simulate database latency
	time.Sleep(time.Duration(rand.Intn(100)+20) * time.Millisecond)

	db.mu.Lock()
	db.data[key] = value
	db.mu.Unlock()

	return nil
}

func (db *DatabaseService) Delete(ctx context.Context, key string) error {
	_, span := tracer.Start(ctx, "db.delete")
	defer span.End()

	span.SetAttributes(attribute.String("key", key))

	if !db.enabled {
		span.RecordError(ErrDatabaseDown)
		span.SetStatus(codes.Error, "Database is disabled")
		return ErrDatabaseDown
	}

	// Simulate database latency
	time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)

	db.mu.Lock()
	delete(db.data, key)
	db.mu.Unlock()

	return nil
}

// CacheService simulates cache operations
type CacheService struct {
	enabled bool
	mu      sync.RWMutex
	data    map[string]interface{}
	ttl     map[string]time.Time
}

func NewCacheService(enabled bool) *CacheService {
	return &CacheService{
		enabled: enabled,
		data:    make(map[string]interface{}),
		ttl:     make(map[string]time.Time),
	}
}

func (c *CacheService) HealthCheck(ctx context.Context) bool {
	_, span := tracer.Start(ctx, "cache.health-check")
	defer span.End()

	// Simulate random health status
	healthy := c.enabled && rand.Float32() > 0.05

	span.SetAttributes(attribute.Bool("healthy", healthy))

	return healthy
}

func (c *CacheService) Get(ctx context.Context, key string) (interface{}, error) {
	_, span := tracer.Start(ctx, "cache.get")
	defer span.End()

	span.SetAttributes(attribute.String("key", key))

	if !c.enabled {
		span.SetAttributes(attribute.Bool("cache.enabled", false))
		return nil, ErrCacheDown
	}

	// Simulate cache latency
	time.Sleep(time.Duration(rand.Intn(5)+1) * time.Millisecond)

	c.mu.RLock()
	value, exists := c.data[key]
	expiry, hasExpiry := c.ttl[key]
	c.mu.RUnlock()

	if !exists {
		span.SetAttributes(attribute.Bool("cache.hit", false))
		return nil, ErrNotFound
	}

	if hasExpiry && time.Now().After(expiry) {
		// Expired
		c.mu.Lock()
		delete(c.data, key)
		delete(c.ttl, key)
		c.mu.Unlock()

		span.SetAttributes(
			attribute.Bool("cache.hit", false),
			attribute.Bool("cache.expired", true),
		)
		return nil, ErrNotFound
	}

	span.SetAttributes(attribute.Bool("cache.hit", true))
	return value, nil
}

func (c *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	_, span := tracer.Start(ctx, "cache.set")
	defer span.End()

	span.SetAttributes(
		attribute.String("key", key),
		attribute.Int64("ttl_seconds", int64(ttl.Seconds())),
	)

	if !c.enabled {
		span.SetAttributes(attribute.Bool("cache.enabled", false))
		return nil // Cache is optional, don't fail
	}

	// Simulate cache latency
	time.Sleep(time.Duration(rand.Intn(5)+1) * time.Millisecond)

	c.mu.Lock()
	c.data[key] = value
	if ttl > 0 {
		c.ttl[key] = time.Now().Add(ttl)
	}
	c.mu.Unlock()

	return nil
}

// UserService handles user-related operations
type UserService struct {
	db    *DatabaseService
	cache *CacheService
}

func NewUserService(db *DatabaseService, cache *CacheService) *UserService {
	return &UserService{
		db:    db,
		cache: cache,
	}
}

func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	ctx, span := tracer.Start(ctx, "service.list-users")
	defer span.End()

	span.SetAttributes(
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)

	// Simulate getting users from database
	users := make([]*User, 0, limit)
	for i := 0; i < limit; i++ {
		users = append(users, &User{
			ID:        fmt.Sprintf("user-%d", offset+i+1),
			Name:      fmt.Sprintf("User %d", offset+i+1),
			Email:     fmt.Sprintf("user%d@example.com", offset+i+1),
			CreatedAt: time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour),
		})
	}

	return users, nil
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
	ctx, span := tracer.Start(ctx, "service.get-user")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	// Try cache first
	cacheKey := fmt.Sprintf("user:%s", userID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		span.SetAttributes(attribute.Bool("cache_hit", true))
		return cached.(*User), nil
	}

	// Get from database
	data, err := s.db.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	user := data.(*User)

	// Cache the result
	s.cache.Set(ctx, cacheKey, user, 5*time.Minute)

	return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
	ctx, span := tracer.Start(ctx, "service.create-user")
	defer span.End()

	user := &User{
		ID:        fmt.Sprintf("user-%d", rand.Intn(10000)),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}

	// Save to database
	key := fmt.Sprintf("user:%s", user.ID)
	if err := s.db.Set(ctx, key, user); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.cache.Set(ctx, key, user, 5*time.Minute)

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID, name, email string) (*User, error) {
	ctx, span := tracer.Start(ctx, "service.update-user")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	// Get existing user
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}

	// Save to database
	key := fmt.Sprintf("user:%s", user.ID)
	if err := s.db.Set(ctx, key, user); err != nil {
		return nil, err
	}

	// Update cache
	s.cache.Set(ctx, key, user, 5*time.Minute)

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	ctx, span := tracer.Start(ctx, "service.delete-user")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	key := fmt.Sprintf("user:%s", userID)
	return s.db.Delete(ctx, key)
}

// ProductService handles product-related operations
type ProductService struct {
	db    *DatabaseService
	cache *CacheService
}

func NewProductService(db *DatabaseService, cache *CacheService) *ProductService {
	return &ProductService{
		db:    db,
		cache: cache,
	}
}

func (s *ProductService) ListProducts(ctx context.Context) ([]*Product, error) {
	ctx, span := tracer.Start(ctx, "service.list-products")
	defer span.End()

	// Simulate getting products from database
	products := []*Product{
		{
			ID:          "prod-1",
			Name:        "Laptop",
			Description: "High-performance laptop",
			Price:       999.99,
			Stock:       50,
		},
		{
			ID:          "prod-2",
			Name:        "Mouse",
			Description: "Wireless mouse",
			Price:       29.99,
			Stock:       200,
		},
		{
			ID:          "prod-3",
			Name:        "Keyboard",
			Description: "Mechanical keyboard",
			Price:       79.99,
			Stock:       100,
		},
	}

	return products, nil
}

func (s *ProductService) GetProduct(ctx context.Context, productID string) (*Product, error) {
	ctx, span := tracer.Start(ctx, "service.get-product")
	defer span.End()

	span.SetAttributes(attribute.String("product_id", productID))

	// Try cache first
	cacheKey := fmt.Sprintf("product:%s", productID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		span.SetAttributes(attribute.Bool("cache_hit", true))
		return cached.(*Product), nil
	}

	// Get from database
	data, err := s.db.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	product := data.(*Product)

	// Cache the result
	s.cache.Set(ctx, cacheKey, product, 10*time.Minute)

	return product, nil
}

func (s *ProductService) CreateProduct(ctx context.Context, name, description string, price float64, stock int) (*Product, error) {
	ctx, span := tracer.Start(ctx, "service.create-product")
	defer span.End()

	product := &Product{
		ID:          fmt.Sprintf("prod-%d", rand.Intn(10000)),
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
	}

	// Save to database
	key := fmt.Sprintf("product:%s", product.ID)
	if err := s.db.Set(ctx, key, product); err != nil {
		return nil, err
	}

	// Cache the product
	s.cache.Set(ctx, key, product, 10*time.Minute)

	return product, nil
}

// OrderService handles order-related operations
type OrderService struct {
	db    *DatabaseService
	cache *CacheService
}

func NewOrderService(db *DatabaseService, cache *CacheService) *OrderService {
	return &OrderService{
		db:    db,
		cache: cache,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID, productID string, quantity int) (*Order, error) {
	ctx, span := tracer.Start(ctx, "service.create-order")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("product_id", productID),
		attribute.Int("quantity", quantity),
	)

	// Verify user exists
	userKey := fmt.Sprintf("user:%s", userID)
	if _, err := s.db.Get(ctx, userKey); err != nil {
		return nil, ErrNotFound
	}

	// Get product and check stock
	productKey := fmt.Sprintf("product:%s", productID)
	data, err := s.db.Get(ctx, productKey)
	if err != nil {
		return nil, ErrNotFound
	}

	product := data.(*Product)
	if product.Stock < quantity {
		return nil, ErrInsufficientStock
	}

	// Create order
	order := &Order{
		ID:         fmt.Sprintf("order-%d", rand.Intn(10000)),
		UserID:     userID,
		ProductID:  productID,
		Quantity:   quantity,
		TotalPrice: product.Price * float64(quantity),
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	// Save order
	orderKey := fmt.Sprintf("order:%s", order.ID)
	if err := s.db.Set(ctx, orderKey, order); err != nil {
		return nil, err
	}

	// Update product stock
	product.Stock -= quantity
	if err := s.db.Set(ctx, productKey, product); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	ctx, span := tracer.Start(ctx, "service.get-order")
	defer span.End()

	span.SetAttributes(attribute.String("order_id", orderID))

	// Try cache first
	cacheKey := fmt.Sprintf("order:%s", orderID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		span.SetAttributes(attribute.Bool("cache_hit", true))
		return cached.(*Order), nil
	}

	// Get from database
	data, err := s.db.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	order := data.(*Order)

	// Cache the result
	s.cache.Set(ctx, cacheKey, order, 5*time.Minute)

	return order, nil
}

// AnalyticsService provides analytics data
type AnalyticsService struct {
	db *DatabaseService
}

func NewAnalyticsService(db *DatabaseService) *AnalyticsService {
	return &AnalyticsService{
		db: db,
	}
}

func (s *AnalyticsService) GetDashboardData(ctx context.Context) (map[string]interface{}, error) {
	ctx, span := tracer.Start(ctx, "service.analytics-dashboard")
	defer span.End()

	// Simulate complex analytics query
	time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)

	// Return mock analytics data
	return map[string]interface{}{
		"total_users":    rand.Intn(10000) + 1000,
		"total_products": rand.Intn(1000) + 100,
		"total_orders":   rand.Intn(50000) + 5000,
		"revenue":        float64(rand.Intn(1000000) + 100000),
		"top_products": []map[string]interface{}{
			{"id": "prod-1", "name": "Laptop", "sales": rand.Intn(1000)},
			{"id": "prod-2", "name": "Mouse", "sales": rand.Intn(5000)},
			{"id": "prod-3", "name": "Keyboard", "sales": rand.Intn(2000)},
		},
		"sales_by_day": generateSalesData(),
		"timestamp":    time.Now().UTC(),
	}, nil
}

func generateSalesData() []map[string]interface{} {
	data := make([]map[string]interface{}, 7)
	for i := 0; i < 7; i++ {
		data[i] = map[string]interface{}{
			"date":  time.Now().AddDate(0, 0, -i).Format("2006-01-02"),
			"sales": rand.Intn(10000) + 1000,
		}
	}
	return data
}

// ExternalAPIService demonstrates circuit breaker pattern
type ExternalAPIService struct {
	breaker *gobreaker.CircuitBreaker
}

func NewExternalAPIService() *ExternalAPIService {
	settings := gobreaker.Settings{
		Name:        "external-api",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	return &ExternalAPIService{
		breaker: gobreaker.NewCircuitBreaker(settings),
	}
}

func (s *ExternalAPIService) CallExternalAPI(ctx context.Context, endpoint string) (interface{}, error) {
	ctx, span := tracer.Start(ctx, "external-api.call")
	defer span.End()

	span.SetAttributes(attribute.String("endpoint", endpoint))

	result, err := s.breaker.Execute(func() (interface{}, error) {
		// Simulate external API call
		_, apiSpan := tracer.Start(ctx, "external-api.request")
		defer apiSpan.End()

		// Simulate random failures
		if rand.Float32() < 0.3 {
			apiSpan.RecordError(ErrExternalAPIDown)
			apiSpan.SetStatus(codes.Error, "External API failed")
			return nil, ErrExternalAPIDown
		}

		// Simulate latency
		latency := time.Duration(rand.Intn(500)+100) * time.Millisecond
		time.Sleep(latency)

		apiSpan.SetAttributes(attribute.Int64("latency_ms", latency.Milliseconds()))

		return map[string]interface{}{
			"status": "success",
			"data":   "External API response",
		}, nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Circuit breaker prevented call or call failed")

		// Check if circuit breaker is open
		if err == gobreaker.ErrOpenState {
			span.SetAttributes(attribute.String("circuit_breaker.state", "open"))
			return nil, fiber.NewError(fiber.StatusServiceUnavailable, "Service temporarily unavailable")
		}

		return nil, err
	}

	return result, nil
}
