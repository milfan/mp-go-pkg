package pkgtelemetry

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

func (t *telemetry) initMetrics(ctx context.Context) error {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(t.config.collectorEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
	}
	t.metricExporter = exporter

	exportInterval := t.config.metricsExportIntervalInSecond
	if exportInterval == 0 {
		exportInterval = 30 * time.Second
	}

	t.meterProvider = metricsdk.NewMeterProvider(
		metricsdk.WithResource(t.resource),
		metricsdk.WithReader(metricsdk.NewPeriodicReader(
			exporter,
			metricsdk.WithInterval(exportInterval),
		)),
	)

	otel.SetMeterProvider(t.meterProvider)

	if t.config.enableRuntime {
		if err := runtime.Start(runtime.WithMeterProvider(t.meterProvider)); err != nil {
			return fmt.Errorf("failed to start runtime metrics: %w", err)
		}
		log.Println("Runtime metrics collection initiated")
	}

	log.Println("Metrics provider initiated")
	return nil
}

func (t *telemetry) MetricsMiddleware(config TelemetryMiddlewareConfig) gin.HandlerFunc {

	meter := otel.Meter(t.config.serviceName)

	metrics := &HttpMetrics{}
	var err error

	metrics.requestCounter, err = meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err)
	}

	metrics.requestDuration, err = meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		panic(err)
	}

	if config.recordRequestSize {
		metrics.requestSize, err = meter.Int64Histogram(
			"http.server.request.size",
			metric.WithDescription("HTTP request content length"),
			metric.WithUnit("By"),
			metric.WithExplicitBucketBoundaries(0, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536),
		)
		if err != nil {
			panic(err)
		}
	}

	if config.recordResponseSize {
		metrics.responseSize, err = meter.Int64Histogram(
			"http.server.response.size",
			metric.WithDescription("HTTP response content length"),
			metric.WithUnit("By"),
			metric.WithExplicitBucketBoundaries(0, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536),
		)
		if err != nil {
			panic(err)
		}
	}

	metrics.activeRequests, err = meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		for _, path := range config.skipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		start := time.Now()
		metrics.activeRequests.Add(c.Request.Context(), 1)
		defer metrics.activeRequests.Add(c.Request.Context(), -1)

		c.Next()

		duration := time.Since(start)
		durationMs := float64(duration.Nanoseconds()) / 1e6

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		attrs := []attribute.KeyValue{
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.String("http.status_class", getStatusClass(c.Writer.Status())),
		}

		metrics.requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
		metrics.requestDuration.Record(c.Request.Context(), durationMs, metric.WithAttributes(attrs...))

		if config.recordRequestSize && metrics.requestSize != nil {
			if c.Request.ContentLength > 0 {
				metrics.requestSize.Record(c.Request.Context(), c.Request.ContentLength, metric.WithAttributes(attrs...))
			}
		}

		if config.recordResponseSize && metrics.responseSize != nil {
			if responseSize := int64(c.Writer.Size()); responseSize > 0 {
				metrics.responseSize.Record(c.Request.Context(), responseSize, metric.WithAttributes(attrs...))
			}
		}
	}
}

func getStatusClass(statusCode int) string {
	class := statusCode / 100
	return strconv.Itoa(class) + "xx"
}
