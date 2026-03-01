package pkgtelemetry

import (
	"time"

	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type TelemetryConfig struct {
	serviceName       string
	collectorEndpoint string
	insecure          bool

	// Metrics
	metricsExportIntervalInSecond time.Duration
	enableRuntime                 bool

	// Tracing
	traceSampleRate float64

	// Middleware Config
	middlewareConfig TelemetryMiddlewareConfig
}

type TelemetryMiddlewareConfig struct {
	skipPaths                []string
	requestIDKey             string
	recordRequestSize        bool
	recordResponseSize       bool
	recordRequestHeaders     bool
	skipRecordRequestHeaders []string
	recordResponseHeaders    bool
}

type HttpMetrics struct {
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	requestSize     metric.Int64Histogram
	responseSize    metric.Int64Histogram
	activeRequests  metric.Int64UpDownCounter
}

type telemetry struct {
	meterProvider  *metricsdk.MeterProvider
	metricExporter metricsdk.Exporter
	tracerProvider *sdktrace.TracerProvider
	traceExporter  sdktrace.SpanExporter
	resource       *resource.Resource
	config         TelemetryConfig
}

type OptFunc func(*TelemetryConfig)

func SetServiceName(serviceName string) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.serviceName = serviceName
	}
}

func SetCollectorEndpoint(endpoint string) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.collectorEndpoint = endpoint
	}
}

func SetInsecure(insecure bool) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.insecure = insecure
	}
}

func SetMetricsExportIntervalInSecond(interval time.Duration) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.metricsExportIntervalInSecond = interval
	}
}

func SetEnableRuntime(enable bool) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.enableRuntime = enable
	}
}

// SetTraceSampleRate sets the trace sampling rate (0.0 to 1.0)
func SetTraceSampleRate(rate float64) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.traceSampleRate = rate
	}
}

func SetMiddlewareConfig(config TelemetryMiddlewareConfig) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig = config
	}
}

// SetSkipPaths sets the paths to skip in middleware
func SetSkipPaths(paths []string) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig.skipPaths = paths
	}
}

func SetRequestIDKey(key string) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig.requestIDKey = key
	}
}

func SetRecordRequestSize(enable bool) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig.recordRequestSize = enable
	}
}

func SetRecordResponseSize(enable bool) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig.recordResponseSize = enable
	}
}

func SetRecordRequestHeaders(enable bool) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig.recordRequestHeaders = enable
	}
}

func SetRecordResponseHeaders(enable bool) OptFunc {
	return func(tc *TelemetryConfig) {
		tc.middlewareConfig.recordResponseHeaders = enable
	}
}
