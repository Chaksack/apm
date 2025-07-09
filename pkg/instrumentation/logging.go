package instrumentation

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerMiddleware returns a Fiber middleware for structured request logging
func LoggerMiddleware(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get request ID if available
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = c.Locals("requestid").(string)
		}

		// Create request-scoped logger
		reqLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		)

		// Store logger in context for use in handlers
		c.Locals("logger", reqLogger)

		// Log request start
		reqLogger.Debug("request started",
			zap.String("host", c.Hostname()),
			zap.String("referer", c.Get("Referer")),
		)

		// Process request
		err := c.Next()

		// Calculate request duration
		duration := time.Since(start)
		status := c.Response().StatusCode()

		// Prepare log fields
		fields := []zapcore.Field{
			zap.Int("status", status),
			zap.Duration("duration", duration),
			zap.Int("bytes_sent", len(c.Response().Body())),
		}

		// Add error if present
		if err != nil {
			fields = append(fields, zap.Error(err))
		}

		// Log based on status code
		switch {
		case status >= 500:
			reqLogger.Error("request failed", fields...)
		case status >= 400:
			reqLogger.Warn("request completed with client error", fields...)
		case status >= 300:
			reqLogger.Info("request redirected", fields...)
		default:
			reqLogger.Info("request completed", fields...)
		}

		return err
	}
}

// GetLogger retrieves the request-scoped logger from Fiber context
func GetLogger(c *fiber.Ctx) *zap.Logger {
	if logger, ok := c.Locals("logger").(*zap.Logger); ok {
		return logger
	}
	// Return a default logger if not found
	return zap.L()
}

// NewLogger creates a new zap logger with the given configuration
func NewLogger(cfg LoggingConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Development {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Set log level
	if err := zapCfg.Level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, err
	}

	// Set output paths
	if len(cfg.OutputPaths) > 0 {
		zapCfg.OutputPaths = cfg.OutputPaths
	}
	if len(cfg.ErrorOutputPaths) > 0 {
		zapCfg.ErrorOutputPaths = cfg.ErrorOutputPaths
	}

	// Set encoding
	if cfg.Encoding != "" {
		zapCfg.Encoding = cfg.Encoding
	}

	// Add custom fields
	if len(cfg.InitialFields) > 0 {
		zapCfg.InitialFields = cfg.InitialFields
	}

	// Build logger
	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	// Add caller info if requested
	if cfg.EnableCaller {
		logger = logger.WithOptions(zap.AddCaller())
	}

	// Add stack trace for error level and above if requested
	if cfg.EnableStacktrace {
		logger = logger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
	}

	return logger, nil
}

// SugaredLogger returns a sugared logger for more convenient logging
func SugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}
