package pkgtelemetry

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type ITelemetry interface {

	// Setup return error if fail to initializes the open telemetry providers componentss
	Setup(ctx context.Context) error

	// MetricsMiddleware return gin.HandlerFunc'
	//
	// It is middleware gin for collecting metrics from incoming request
	MetricsMiddleware(config TelemetryMiddlewareConfig) gin.HandlerFunc

	// TracingMiddleware return gin.HandlerFunc'
	//
	// It is middleware gin for automatically create span trace record for incoming request
	TracingMiddleware(config TelemetryMiddlewareConfig) gin.HandlerFunc

	// InitAutoTraceGorm auto tracing for gorm request database
	//
	// This function automatically create span trace for gorm request to database
	InitAutoTraceGorm(connectionName string, gormDB *gorm.DB) error

	// Shutdown returning error if stoping meter provider trace and metrics are fail.
	Shutdown(ctx context.Context) error

	// GetMeterProvider returning *metricsdk.MeterProvider
	// use to create new meter for custom metrics
	GetMeterProvider() *metricsdk.MeterProvider

	// GetMeterProvider returning *metricsdk.MeterProvider
	GetTracerProvider() *sdktrace.TracerProvider

	// TelemetryConfig returning TelemetryConfig
	GetConfig() TelemetryConfig

	// GetMiddlewareConfig returning TelemetryMiddlewareConfig that used for
	// auto collect data metrics and trace
	GetMiddlewareConfig() TelemetryMiddlewareConfig

	// NewSpan returning span context and trace span
	//
	// It is a helper to create new span.
	//
	// Accept gin.Context, map[string]interface{}
	//  - Usage: telemetry.NewSpan(ctx, "user.id", map[string]interface{}{"user.id": "AHS2025"})
	NewSpan(c *gin.Context, spanName string, attributes map[string]interface{}) (context.Context, trace.Span)

	// AddSpan returning  trace span
	//
	// It is a helper to add span. Used to create new child span
	//
	// Accept *context.Context, map[string]interface{}
	//  - Usage: telemetry.AddSpan(ctx, "user.id", map[string]interface{}{"user.id": "AHS2025"})
	AddSpan(ctx *context.Context, spanName string, attributes map[string]interface{}) trace.Span

	// AddSpanAttributes is a helper to add attributes to the current span
	//
	// If no Span is currently set in ctx, that performs no operation returned
	// It's set attribute to noopSpanInstance
	//
	//  - Usage: telemetry.AddSpanAttributes(ctx, "user.id", map[string]interface{}{"user.id": "AHS2025"})
	AddSpanAttributes(ctx context.Context, attributes map[string]interface{})

	// AddSpanEvent is a helper to add events to the current span
	//
	//   - Usage: telemetry.AddSpanEvent(ctx, "cache.hit", map[string]interface{}{"user.id": "AHS2025"})
	AddSpanEvent(ctx context.Context, name string, attributes map[string]interface{})

	// RecordError is a helper to record an error in the current span
	//
	//   - Usage: telemetry.RecordError(ctx, err, map[string]interface{}{"user.id": "AHS2025"})
	RecordError(ctx context.Context, err error, attributes map[string]interface{})
}

// NewTelemetry creates a new telemetry instance
//
// With minimal config: NewTelemetry(SetServiceName("my-service"), SetCollectorEndpoint("localhost:4318"))
//
// To config auto trace and metrics collection middleware use pkgtelemetry.SetCollectorEndpoint(config)
func NewTelemetry(opts ...OptFunc) ITelemetry {
	conf := TelemetryConfig{
		insecure:                      true,
		metricsExportIntervalInSecond: 30 * time.Second,
		enableRuntime:                 true,
		traceSampleRate:               1.0,
		middlewareConfig: TelemetryMiddlewareConfig{
			skipPaths:                []string{"/health", "/metrics", "/ready"},
			skipRecordRequestHeaders: []string{"Authorization"},
			requestIDKey:             "request_id",
			recordRequestSize:        true,
			recordResponseSize:       true,
			recordRequestHeaders:     false,
			recordResponseHeaders:    false,
		},
	}

	for _, opt := range opts {
		opt(&conf)
	}

	telemetry := telemetry{
		config: conf,
	}

	return &telemetry
}

func (t *telemetry) Setup(ctx context.Context) error {
	if err := validateAndSetDefaults(&t.config); err != nil {
		return fmt.Errorf("invalid telemetry config: %w", err)
	}

	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(t.config.serviceName),
			semconv.ServiceInstanceID(host),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}
	t.resource = res

	if err := t.initMetrics(ctx); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	if err := t.initTracing(ctx); err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	return nil
}

func (t *telemetry) GetMeterProvider() *metricsdk.MeterProvider {
	return t.meterProvider
}

func (t *telemetry) GetTracerProvider() *sdktrace.TracerProvider {
	return t.tracerProvider
}

func (t *telemetry) GetConfig() TelemetryConfig {
	return t.config
}

func (t *telemetry) GetMiddlewareConfig() TelemetryMiddlewareConfig {
	return t.config.middlewareConfig
}

func (t *telemetry) Shutdown(ctx context.Context) error {
	var errs []error

	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		}
	}

	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("telemetry shutdown errors: %v", errs)
	}

	log.Println("Telemetry shutdown completed")
	return nil
}

// validateAndSetDefaults validates the config and sets default values
func validateAndSetDefaults(config *TelemetryConfig) error {
	if config.serviceName == "" {
		return fmt.Errorf("ServiceName is required")
	}

	if config.collectorEndpoint == "" {
		return fmt.Errorf("CollectorEndpoint is required")
	}

	if config.metricsExportIntervalInSecond == 0 {
		config.metricsExportIntervalInSecond = 30 * time.Second
	}

	if config.traceSampleRate == 0 {
		config.traceSampleRate = 1.0 // 100% sampling by default
	}

	// Validate sample rate is between 0 and 1
	if config.traceSampleRate < 0 || config.traceSampleRate > 1 {
		return fmt.Errorf("TraceSampleRate must be between 0 and 1, got: %f", config.traceSampleRate)
	}

	if config.middlewareConfig.skipPaths == nil {
		config.middlewareConfig.skipPaths = []string{"/health", "/metrics", "/ready"}
	}

	if config.middlewareConfig.skipRecordRequestHeaders == nil {
		config.middlewareConfig.skipRecordRequestHeaders = []string{"Authorization"}
	}

	return nil
}
