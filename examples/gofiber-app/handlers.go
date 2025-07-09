package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// User represents a user entity
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Product represents a product entity
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// Order represents an order entity
type Order struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ProductID  string    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// HealthHandler returns the health check handler
func NewHealthHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "health-check")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Check database health
		dbHealthy := true
		if services.db != nil {
			dbHealthy = services.db.HealthCheck(ctx)
		}

		// Check cache health
		cacheHealthy := true
		if services.cache != nil {
			cacheHealthy = services.cache.HealthCheck(ctx)
		}

		// Overall health status
		healthy := dbHealthy && cacheHealthy
		status := "healthy"
		if !healthy {
			status = "unhealthy"
		}

		span.SetAttributes(
			attribute.Bool("db.healthy", dbHealthy),
			attribute.Bool("cache.healthy", cacheHealthy),
			attribute.String("status", status),
		)

		logger.Info("Health check performed",
			zap.String("status", status),
			zap.Bool("db_healthy", dbHealthy),
			zap.Bool("cache_healthy", cacheHealthy),
		)

		response := fiber.Map{
			"status":    status,
			"timestamp": time.Now().UTC(),
			"service":   appConfig.AppName,
			"version":   appConfig.Version,
			"checks": fiber.Map{
				"database": fiber.Map{
					"status": boolToStatus(dbHealthy),
				},
				"cache": fiber.Map{
					"status": boolToStatus(cacheHealthy),
				},
			},
		}

		if !healthy {
			return c.Status(fiber.StatusServiceUnavailable).JSON(response)
		}

		return c.JSON(response)
	}
}

// ListUsersHandler returns a list of users
func NewListUsersHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "list-users")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Parse query parameters
		limit := c.QueryInt("limit", 10)
		offset := c.QueryInt("offset", 0)

		span.SetAttributes(
			attribute.Int("limit", limit),
			attribute.Int("offset", offset),
		)

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("list_users"))
		defer timer.ObserveDuration()

		// Get users from service
		users, err := services.user.ListUsers(ctx, limit, offset)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to list users")
			businessMetrics.WithLabelValues("list_users", "error").Inc()

			logger.Error("Failed to list users", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list users")
		}

		businessMetrics.WithLabelValues("list_users", "success").Inc()

		logger.Info("Users listed successfully",
			zap.Int("count", len(users)),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
		)

		return c.JSON(fiber.Map{
			"users":  users,
			"count":  len(users),
			"limit":  limit,
			"offset": offset,
		})
	}
}

// GetUserHandler returns a specific user
func NewGetUserHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "get-user")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)
		userID := c.Params("id")

		span.SetAttributes(attribute.String("user.id", userID))

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("get_user"))
		defer timer.ObserveDuration()

		// Get user from service
		user, err := services.user.GetUser(ctx, userID)
		if err != nil {
			if err == ErrNotFound {
				span.SetStatus(codes.Error, "User not found")
				businessMetrics.WithLabelValues("get_user", "not_found").Inc()

				logger.Warn("User not found", zap.String("user_id", userID))
				return fiber.NewError(fiber.StatusNotFound, "User not found")
			}

			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get user")
			businessMetrics.WithLabelValues("get_user", "error").Inc()

			logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user")
		}

		businessMetrics.WithLabelValues("get_user", "success").Inc()

		logger.Info("User retrieved successfully", zap.String("user_id", userID))

		return c.JSON(user)
	}
}

// CreateUserHandler creates a new user
func NewCreateUserHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "create-user")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Parse request body
		var req struct {
			Name  string `json:"name" validate:"required"`
			Email string `json:"email" validate:"required,email"`
		}

		if err := c.BodyParser(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			businessMetrics.WithLabelValues("create_user", "invalid_request").Inc()

			logger.Error("Invalid request body", zap.Error(err))
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		span.SetAttributes(
			attribute.String("user.name", req.Name),
			attribute.String("user.email", req.Email),
		)

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("create_user"))
		defer timer.ObserveDuration()

		// Create user via service
		user, err := services.user.CreateUser(ctx, req.Name, req.Email)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create user")
			businessMetrics.WithLabelValues("create_user", "error").Inc()

			logger.Error("Failed to create user", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create user")
		}

		businessMetrics.WithLabelValues("create_user", "success").Inc()

		logger.Info("User created successfully",
			zap.String("user_id", user.ID),
			zap.String("email", user.Email),
		)

		return c.Status(fiber.StatusCreated).JSON(user)
	}
}

// UpdateUserHandler updates an existing user
func NewUpdateUserHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "update-user")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)
		userID := c.Params("id")

		span.SetAttributes(attribute.String("user.id", userID))

		// Parse request body
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		if err := c.BodyParser(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			businessMetrics.WithLabelValues("update_user", "invalid_request").Inc()

			logger.Error("Invalid request body", zap.Error(err))
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("update_user"))
		defer timer.ObserveDuration()

		// Update user via service
		user, err := services.user.UpdateUser(ctx, userID, req.Name, req.Email)
		if err != nil {
			if err == ErrNotFound {
				span.SetStatus(codes.Error, "User not found")
				businessMetrics.WithLabelValues("update_user", "not_found").Inc()

				logger.Warn("User not found", zap.String("user_id", userID))
				return fiber.NewError(fiber.StatusNotFound, "User not found")
			}

			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to update user")
			businessMetrics.WithLabelValues("update_user", "error").Inc()

			logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", userID))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update user")
		}

		businessMetrics.WithLabelValues("update_user", "success").Inc()

		logger.Info("User updated successfully", zap.String("user_id", userID))

		return c.JSON(user)
	}
}

// DeleteUserHandler deletes a user
func NewDeleteUserHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "delete-user")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)
		userID := c.Params("id")

		span.SetAttributes(attribute.String("user.id", userID))

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("delete_user"))
		defer timer.ObserveDuration()

		// Delete user via service
		err := services.user.DeleteUser(ctx, userID)
		if err != nil {
			if err == ErrNotFound {
				span.SetStatus(codes.Error, "User not found")
				businessMetrics.WithLabelValues("delete_user", "not_found").Inc()

				logger.Warn("User not found", zap.String("user_id", userID))
				return fiber.NewError(fiber.StatusNotFound, "User not found")
			}

			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to delete user")
			businessMetrics.WithLabelValues("delete_user", "error").Inc()

			logger.Error("Failed to delete user", zap.Error(err), zap.String("user_id", userID))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete user")
		}

		businessMetrics.WithLabelValues("delete_user", "success").Inc()

		logger.Info("User deleted successfully", zap.String("user_id", userID))

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ListProductsHandler returns a list of products
func NewListProductsHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "list-products")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("list_products"))
		defer timer.ObserveDuration()

		// Get products from service
		products, err := services.product.ListProducts(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to list products")
			businessMetrics.WithLabelValues("list_products", "error").Inc()

			logger.Error("Failed to list products", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list products")
		}

		businessMetrics.WithLabelValues("list_products", "success").Inc()

		logger.Info("Products listed successfully", zap.Int("count", len(products)))

		return c.JSON(fiber.Map{
			"products": products,
			"count":    len(products),
		})
	}
}

// GetProductHandler returns a specific product
func NewGetProductHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "get-product")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)
		productID := c.Params("id")

		span.SetAttributes(attribute.String("product.id", productID))

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("get_product"))
		defer timer.ObserveDuration()

		// Get product from service
		product, err := services.product.GetProduct(ctx, productID)
		if err != nil {
			if err == ErrNotFound {
				span.SetStatus(codes.Error, "Product not found")
				businessMetrics.WithLabelValues("get_product", "not_found").Inc()

				logger.Warn("Product not found", zap.String("product_id", productID))
				return fiber.NewError(fiber.StatusNotFound, "Product not found")
			}

			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get product")
			businessMetrics.WithLabelValues("get_product", "error").Inc()

			logger.Error("Failed to get product", zap.Error(err), zap.String("product_id", productID))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get product")
		}

		businessMetrics.WithLabelValues("get_product", "success").Inc()

		logger.Info("Product retrieved successfully", zap.String("product_id", productID))

		return c.JSON(product)
	}
}

// CreateProductHandler creates a new product
func NewCreateProductHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "create-product")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Parse request body
		var req struct {
			Name        string  `json:"name" validate:"required"`
			Description string  `json:"description"`
			Price       float64 `json:"price" validate:"required,min=0"`
			Stock       int     `json:"stock" validate:"required,min=0"`
		}

		if err := c.BodyParser(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			businessMetrics.WithLabelValues("create_product", "invalid_request").Inc()

			logger.Error("Invalid request body", zap.Error(err))
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		span.SetAttributes(
			attribute.String("product.name", req.Name),
			attribute.Float64("product.price", req.Price),
			attribute.Int("product.stock", req.Stock),
		)

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("create_product"))
		defer timer.ObserveDuration()

		// Create product via service
		product, err := services.product.CreateProduct(ctx, req.Name, req.Description, req.Price, req.Stock)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create product")
			businessMetrics.WithLabelValues("create_product", "error").Inc()

			logger.Error("Failed to create product", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create product")
		}

		businessMetrics.WithLabelValues("create_product", "success").Inc()

		logger.Info("Product created successfully",
			zap.String("product_id", product.ID),
			zap.String("name", product.Name),
		)

		return c.Status(fiber.StatusCreated).JSON(product)
	}
}

// CreateOrderHandler creates a new order
func NewCreateOrderHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "create-order")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Parse request body
		var req struct {
			UserID    string `json:"user_id" validate:"required"`
			ProductID string `json:"product_id" validate:"required"`
			Quantity  int    `json:"quantity" validate:"required,min=1"`
		}

		if err := c.BodyParser(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			businessMetrics.WithLabelValues("create_order", "invalid_request").Inc()

			logger.Error("Invalid request body", zap.Error(err))
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		span.SetAttributes(
			attribute.String("order.user_id", req.UserID),
			attribute.String("order.product_id", req.ProductID),
			attribute.Int("order.quantity", req.Quantity),
		)

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("create_order"))
		defer timer.ObserveDuration()

		// Create order via service
		order, err := services.order.CreateOrder(ctx, req.UserID, req.ProductID, req.Quantity)
		if err != nil {
			if err == ErrNotFound {
				span.SetStatus(codes.Error, "User or product not found")
				businessMetrics.WithLabelValues("create_order", "not_found").Inc()

				logger.Warn("User or product not found")
				return fiber.NewError(fiber.StatusNotFound, "User or product not found")
			}

			if err == ErrInsufficientStock {
				span.SetStatus(codes.Error, "Insufficient stock")
				businessMetrics.WithLabelValues("create_order", "insufficient_stock").Inc()

				logger.Warn("Insufficient stock", zap.String("product_id", req.ProductID))
				return fiber.NewError(fiber.StatusBadRequest, "Insufficient stock")
			}

			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create order")
			businessMetrics.WithLabelValues("create_order", "error").Inc()

			logger.Error("Failed to create order", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create order")
		}

		businessMetrics.WithLabelValues("create_order", "success").Inc()

		logger.Info("Order created successfully",
			zap.String("order_id", order.ID),
			zap.String("user_id", order.UserID),
			zap.String("product_id", order.ProductID),
			zap.Float64("total_price", order.TotalPrice),
		)

		return c.Status(fiber.StatusCreated).JSON(order)
	}
}

// GetOrderHandler returns a specific order
func NewGetOrderHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "get-order")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)
		orderID := c.Params("id")

		span.SetAttributes(attribute.String("order.id", orderID))

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("get_order"))
		defer timer.ObserveDuration()

		// Get order from service
		order, err := services.order.GetOrder(ctx, orderID)
		if err != nil {
			if err == ErrNotFound {
				span.SetStatus(codes.Error, "Order not found")
				businessMetrics.WithLabelValues("get_order", "not_found").Inc()

				logger.Warn("Order not found", zap.String("order_id", orderID))
				return fiber.NewError(fiber.StatusNotFound, "Order not found")
			}

			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get order")
			businessMetrics.WithLabelValues("get_order", "error").Inc()

			logger.Error("Failed to get order", zap.Error(err), zap.String("order_id", orderID))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get order")
		}

		businessMetrics.WithLabelValues("get_order", "success").Inc()

		logger.Info("Order retrieved successfully", zap.String("order_id", orderID))

		return c.JSON(order)
	}
}

// AnalyticsDashboardHandler returns analytics dashboard data
func NewAnalyticsDashboardHandler(services *Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "analytics-dashboard")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Measure operation duration
		timer := prometheus.NewTimer(operationDuration.WithLabelValues("analytics_dashboard"))
		defer timer.ObserveDuration()

		// Get analytics data from service
		analytics, err := services.analytics.GetDashboardData(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get analytics")
			businessMetrics.WithLabelValues("analytics_dashboard", "error").Inc()

			logger.Error("Failed to get analytics", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get analytics")
		}

		businessMetrics.WithLabelValues("analytics_dashboard", "success").Inc()

		logger.Info("Analytics retrieved successfully")

		return c.JSON(analytics)
	}
}

// Test handlers for demonstrating different scenarios

// SlowHandler simulates a slow endpoint
func NewSlowHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "slow-endpoint")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Random delay between 1-5 seconds
		delay := time.Duration(rand.Intn(4000)+1000) * time.Millisecond

		span.SetAttributes(attribute.Int64("delay_ms", delay.Milliseconds()))

		logger.Info("Slow operation started", zap.Duration("delay", delay))

		time.Sleep(delay)

		businessMetrics.WithLabelValues("slow_operation", "success").Inc()

		return c.JSON(fiber.Map{
			"message":  "Slow operation completed",
			"delay_ms": delay.Milliseconds(),
		})
	}
}

// ErrorHandler simulates random errors
func NewErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "error-endpoint")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		// Random error code
		errorCodes := []int{
			fiber.StatusBadRequest,
			fiber.StatusNotFound,
			fiber.StatusInternalServerError,
			fiber.StatusServiceUnavailable,
		}

		code := errorCodes[rand.Intn(len(errorCodes))]

		span.SetAttributes(attribute.Int("error_code", code))
		span.SetStatus(codes.Error, fmt.Sprintf("Simulated error with code %d", code))

		businessMetrics.WithLabelValues("error_test", "error").Inc()

		logger.Error("Simulated error", zap.Int("code", code))

		return fiber.NewError(code, fmt.Sprintf("Simulated error with code %d", code))
	}
}

// PanicHandler simulates a panic (will be recovered by middleware)
func NewPanicHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := tracer.Start(c.UserContext(), "panic-endpoint")
		defer span.End()

		logger := c.Locals("logger").(*zap.Logger)

		logger.Error("About to panic!")

		span.SetStatus(codes.Error, "Panic simulated")

		panic("Simulated panic for testing recovery")
	}
}

// Helper functions

func boolToStatus(b bool) string {
	if b {
		return "healthy"
	}
	return "unhealthy"
}
