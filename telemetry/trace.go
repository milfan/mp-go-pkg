package pkgtelemetry

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func (t *telemetry) initTracing(ctx context.Context) error {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(t.config.collectorEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}
	t.traceExporter = exporter

	sampleRate := t.config.traceSampleRate
	if sampleRate == 0 {
		sampleRate = 1.0
	}

	t.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(t.resource),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(
			sdktrace.ParentBased(
				sdktrace.TraceIDRatioBased(sampleRate),
			),
		),
	)

	otel.SetTracerProvider(t.tracerProvider)

	log.Println("Trace provider initiated")
	return nil
}

func (t *telemetry) TracingMiddleware(config TelemetryMiddlewareConfig) gin.HandlerFunc {
	tracer := otel.Tracer(t.config.serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx *gin.Context) {
		for _, path := range config.skipPaths {
			if ctx.Request.URL.Path == path {
				ctx.Next()
				return
			}
		}

		carrier := propagation.HeaderCarrier(ctx.Request.Header)
		parentCtx := propagator.Extract(ctx.Request.Context(), carrier)

		spanName := fmt.Sprintf("%s %s", ctx.Request.Method, ctx.FullPath())
		if ctx.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", ctx.Request.Method, ctx.Request.URL.Path)
		}

		host, err := os.Hostname()
		if err != nil {
			host = "unknown"
		}

		spanCtx, span := tracer.Start(
			parentCtx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", ctx.Request.Method),
				attribute.String("http.url", ctx.Request.URL.String()),
				attribute.String("http.target", ctx.Request.URL.Path),
				attribute.String("http.client_ip", ctx.ClientIP()),
				attribute.String("user_agent", ctx.Request.UserAgent()),
				attribute.String("host", host),
			),
		)
		defer span.End()

		if config.recordRequestHeaders {
			skip := make(map[string]struct{}, len(t.GetMiddlewareConfig().skipRecordRequestHeaders))
			for _, h := range t.GetMiddlewareConfig().skipRecordRequestHeaders {
				skip[strings.ToLower(h)] = struct{}{}
			}

			for key, values := range ctx.Request.Header {
				if _, found := skip[strings.ToLower(key)]; found {
					continue
				}
				if len(values) > 0 {
					span.SetAttributes(attribute.String(fmt.Sprintf("http.request.header.%s", key), values[0]))
				}
			}
		}

		ctx.Request = ctx.Request.WithContext(spanCtx)
		propagator.Inject(spanCtx, propagation.HeaderCarrier(ctx.Writer.Header()))

		ctx.Next()

		statusCode := ctx.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_content_length", ctx.Writer.Size()),
		)

		if config.recordResponseHeaders {
			for key, values := range ctx.Writer.Header() {
				if len(values) > 0 {
					span.SetAttributes(attribute.String(fmt.Sprintf("http.response.header.%s", key), values[0]))
				}
			}
		}

		if config.requestIDKey != "" {
			if requestID, exists := ctx.Get(config.requestIDKey); exists {
				if reqID, ok := requestID.(string); ok {
					span.SetAttributes(attribute.String("request.id", reqID))
				}
			}
		}

		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_content_length", ctx.Writer.Size()),
		)

		if len(ctx.Errors) > 0 {
			lastErr := ctx.Errors.Last().Err

			span.RecordError(lastErr)
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))

			if v, ok := ctx.Get(AppErrorCode); ok {
				if s, ok := v.(string); ok && s != "" {
					span.SetAttributes(attribute.String("app.error.code", s))
				}
			}
			if v, ok := ctx.Get(AppErrorMessage); ok {
				if s, ok := v.(string); ok && s != "" {
					span.SetAttributes(attribute.String("app.error.message", s))
				}
			}
			if v, ok := ctx.Get(AppErrorTrace); ok {
				if s, ok := v.(string); ok && s != "" {
					span.SetAttributes(attribute.String("app.error.error_trace", s))
				}
			}
			if v, ok := ctx.Get(AppErrorRequest); ok {
				if s, ok := v.(string); ok && s != "" {
					span.SetAttributes(attribute.String("app.error.error_request", s))
				}
			}
		}

		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

func (t *telemetry) InitAutoTraceGorm(connectionName string, gormDB *gorm.DB) error {
	err := gormDB.Use(otelgorm.NewPlugin(
		otelgorm.WithDBName(connectionName),
		otelgorm.WithAttributes(
			semconv.DBSystemMSSQL,
		),

		otelgorm.WithTracerProvider(otel.GetTracerProvider()),
	))

	if err != nil {
		return err
	}

	log.Println("Auto trace gorm " + connectionName + " initiated")

	return nil
}

func (t *telemetry) NewSpan(c *gin.Context, spanName string, attributes map[string]interface{}) (context.Context, trace.Span) {
	tracer := otel.Tracer("app")
	newCtx, span := tracer.Start(c.Request.Context(), spanName)

	spanAttr := t.createSpanAttributes(attributes)
	span.SetAttributes(spanAttr...)

	c.Request = c.Request.WithContext(newCtx)

	return c.Request.Context(), span
}

func (t *telemetry) AddSpan(ctx *context.Context, spanName string, attributes map[string]interface{}) trace.Span {
	tracer := otel.Tracer("app")
	newCtx, span := tracer.Start(*ctx, spanName)

	spanAttr := t.createSpanAttributes(attributes)
	span.SetAttributes(spanAttr...)

	*ctx = newCtx

	return span
}

func (t *telemetry) AddSpanAttributes(ctx context.Context, attributes map[string]interface{}) {
	span := trace.SpanFromContext(ctx)

	spanAttr := t.createSpanAttributes(attributes)
	span.SetAttributes(spanAttr...)
}

func (t *telemetry) AddSpanEvent(ctx context.Context, name string, attributes map[string]interface{}) {
	span := trace.SpanFromContext(ctx)

	spanAttr := t.createSpanAttributes(attributes)
	span.SetAttributes(spanAttr...)

	span.AddEvent(name)
}

func (t *telemetry) RecordError(ctx context.Context, err error, attributes map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)

	spanAttr := t.createSpanAttributes(attributes)
	span.SetAttributes(spanAttr...)

	span.SetStatus(codes.Error, err.Error())
}

// createSpanAttributes converts a map to OpenTelemetry attributes
func (t *telemetry) createSpanAttributes(attr map[string]interface{}) []attribute.KeyValue {
	attributes := make([]attribute.KeyValue, 0, len(attr))

	for key, value := range attr {
		switch v := value.(type) {
		case string:
			attributes = append(attributes, attribute.String(key, v))
		case int:
			attributes = append(attributes, attribute.Int(key, v))
		case int64:
			attributes = append(attributes, attribute.Int64(key, v))
		case float64:
			attributes = append(attributes, attribute.Float64(key, v))
		case bool:
			attributes = append(attributes, attribute.Bool(key, v))
		case []string:
			attributes = append(attributes, attribute.StringSlice(key, v))
		case []int:
			attributes = append(attributes, attribute.IntSlice(key, v))
		case []int64:
			attributes = append(attributes, attribute.Int64Slice(key, v))
		case []float64:
			attributes = append(attributes, attribute.Float64Slice(key, v))
		case []bool:
			attributes = append(attributes, attribute.BoolSlice(key, v))
		default:
			// For unsupported types, convert to string
			attributes = append(attributes, attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}

	return attributes
}
