// DevSecOps Demo — Go Backend (Production Variant)
//
// Gin-based REST API with GORM, full OpenTelemetry instrumentation,
// and Pyroscope continuous profiling. API-compatible with the Python backend.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grafana/pyroscope-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const (
	serviceName    = "backend"
	serviceVersion = "1.0.0"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Data Model
// ---------------------------------------------------------------------------

// Item is the core domain object.
type Item struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string `gorm:"index" json:"title"`
	Description string `json:"description"`
}

// ---------------------------------------------------------------------------
// OpenTelemetry Setup
// ---------------------------------------------------------------------------

var (
	tracer              trace.Tracer
	itemsCreatedCounter otelmetric.Int64Counter
)

func initTelemetry(ctx context.Context) (func(), error) {
	endpoint := envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "alloy:4317")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// --- Traces (10 % sampling) ---
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)),
	)
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer(serviceName)

	// --- Metrics ---
	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExp, metric.WithInterval(15*time.Second))),
	)
	otel.SetMeterProvider(mp)

	meter := mp.Meter(serviceName)
	itemsCreatedCounter, err = meter.Int64Counter("items_created_total",
		otelmetric.WithDescription("Total number of items created"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("create counter: %w", err)
	}

	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutdownCtx)
		_ = mp.Shutdown(shutdownCtx)
	}

	return shutdown, nil
}

func initPyroscope() {
	server := envOrDefault("PYROSCOPE_SERVER", "http://alloy:4040")
	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: serviceName,
		ServerAddress:   server,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
		},
	})
	if err != nil {
		slog.Warn("Pyroscope not available — skipping profiling", "error", err)
	} else {
		slog.Info("Pyroscope profiling enabled", "server", server)
	}
}

// ---------------------------------------------------------------------------
// Database
// ---------------------------------------------------------------------------

func initDB() *gorm.DB {
	dsn := envOrDefault("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=items port=5432 sslmode=disable")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// OTel instrumentation for GORM
	if err := db.Use(tracing.NewPlugin()); err != nil {
		slog.Warn("Failed to instrument GORM with OTel", "error", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&Item{}); err != nil {
		slog.Error("Failed to migrate database", "error", err)
		os.Exit(1)
	}

	slog.Info("Database connected and migrated")
	return db
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyzHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

func createItemHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var item Item
		if err := c.ShouldBindJSON(&item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create item"})
			return
		}

		if itemsCreatedCounter != nil {
			itemsCreatedCounter.Add(c.Request.Context(), 1,
				otelmetric.WithAttributes(attribute.String("status", "success")),
			)
		}

		slog.Info("Item created", "id", item.ID, "title", item.Title)
		c.JSON(http.StatusCreated, item)
	}
}

func listItemsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

		ctx := c.Request.Context()
		var items []Item

		if tracer != nil {
			ctx, span := tracer.Start(ctx, "fetch_all_items_from_db",
				trace.WithAttributes(
					attribute.Int("db.skip", skip),
					attribute.Int("db.limit", limit),
				),
			)
			defer span.End()
			db.WithContext(ctx).Offset(skip).Limit(limit).Find(&items)
		} else {
			db.WithContext(ctx).Offset(skip).Limit(limit).Find(&items)
		}

		c.JSON(http.StatusOK, items)
	}
}

func updateItemHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("item_id")

		var existing Item
		if err := db.WithContext(c.Request.Context()).First(&existing, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Item not found"})
			return
		}

		var update Item
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		existing.Title = update.Title
		existing.Description = update.Description
		db.WithContext(c.Request.Context()).Save(&existing)

		slog.Info("Item updated", "id", existing.ID)
		c.JSON(http.StatusOK, existing)
	}
}

func deleteItemHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("item_id")

		var item Item
		if err := db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Item not found"})
			return
		}

		db.WithContext(c.Request.Context()).Delete(&item)
		slog.Info("Item deleted", "id", id)
		c.Status(http.StatusNoContent)
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	ctx := context.Background()

	enableObs := envOrDefault("ENABLE_OBSERVABILITY", "false")
	if enableObs == "true" {
		shutdown, err := initTelemetry(ctx)
		if err != nil {
			slog.Error("Failed to init telemetry", "error", err)
		} else {
			defer shutdown()
			slog.Info("OpenTelemetry instrumentation initialised")
		}
		initPyroscope()
	}

	db := initDB()

	// --- Gin Router ---
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS
	allowedOrigin := envOrDefault("ALLOWED_ORIGIN", "http://localhost:8080")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "traceparent", "tracestate"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- Routes ---
	r.GET("/healthz", healthzHandler)
	r.GET("/readyz", readyzHandler(db))
	r.POST("/items", createItemHandler(db))
	r.GET("/items", listItemsHandler(db))
	r.PUT("/items/:item_id", updateItemHandler(db))
	r.DELETE("/items/:item_id", deleteItemHandler(db))

	// --- Graceful Shutdown ---
	port := envOrDefault("PORT", "8000")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("Server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server…")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced shutdown", "error", err)
	}
	slog.Info("Server stopped")
}
