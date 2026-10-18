package middleware

import (
        "net/http"
        "strconv"
        "time"

        "github.com/ecommerce/backend/pkg/telemetry"
        "github.com/gofiber/fiber/v2"
        "go.opentelemetry.io/otel"
        "go.opentelemetry.io/otel/attribute"
        "go.opentelemetry.io/otel/baggage"
        "go.opentelemetry.io/otel/codes"
        "go.opentelemetry.io/otel/trace"
)

// TelemetryConfig holds references for the telemetry middleware.
type TelemetryConfig struct {
        HTTPMetrics *telemetry.HTTPMetrics
}

// fiberHeaderCarrier adapts Fiber's *fasthttp.RequestCtx to OTel's TextMapCarrier.
type fiberHeaderCarrier struct {
        c *fiber.Ctx
}

func (f *fiberHeaderCarrier) Get(key string) string {
        return f.c.Get(key)
}

func (f *fiberHeaderCarrier) Set(key, value string) {
        f.c.Set(key, value)
}

func (f *fiberHeaderCarrier) Keys() []string {
        headers := f.c.GetReqHeaders()
        keys := make([]string, 0, len(headers))
        for k := range headers {
                keys = append(keys, k)
        }
        return keys
}

// Telemetry returns Fiber middleware that:
//   - Extracts W3C TraceContext (traceparent/tracestate) from incoming headers
//   - Creates an OTel span for each HTTP request as a child of the incoming trace
//   - Extracts W3C Baggage for cross-service attribute propagation
//   - Records HTTP request count, latency, and error metrics
//   - Propagates request_id, trace_id, correlation_id as span attributes
//   - Injects trace context into response headers for distributed tracing
func Telemetry(cfg TelemetryConfig) fiber.Handler {
        tracer := otel.Tracer("github.com/ecommerce/backend/internal/middleware")
        propagator := otel.GetTextMapPropagator()

        return func(c *fiber.Ctx) error {
                startTime := time.Now()

                // Extract incoming trace context from W3C traceparent/tracestate headers.
                carrier := &fiberHeaderCarrier{c: c}
                ctx := propagator.Extract(c.UserContext(), carrier)

                // Also extract W3C Baggage for cross-service propagation.
                ctx = baggage.ContextWithBaggage(ctx, baggage.FromContext(ctx))

                // Build span attributes from the request (no secrets exposed).
                attrs := []attribute.KeyValue{
                        attribute.String("http.method", c.Method()),
                        attribute.String("http.route", c.Route().Path),
                        attribute.String("http.scheme", c.Protocol()),
                        attribute.String("http.host", c.Hostname()),
                        attribute.String("http.target", c.Path()),
                        attribute.Int("http.status_code", http.StatusOK),
                }

                // Propagate request_id from locals (set by RequestID middleware).
                if requestID, ok := c.Locals("request_id").(string); ok {
                        attrs = append(attrs, attribute.String("request_id", requestID))
                }

                spanName := c.Method() + " " + c.Route().Path
                ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
                defer span.End()

                // Store the span context in Fiber locals so downstream handlers can access it.
                c.Locals("otel_span_ctx", ctx)
                c.SetUserContext(ctx)

                // Process the request.
                err := c.Next()

                // Record response attributes.
                status := c.Response().StatusCode()
                span.SetAttributes(
                        attribute.Int("http.status_code", status),
                )
                if err != nil {
                        telemetry.RecordSpanError(span, err)
                }
                if status >= 500 {
                        span.SetStatus(codes.Error, "server error")
                }

                // Inject trace context into response headers for client-side correlation.
                sc := trace.SpanContextFromContext(ctx)
                if sc.IsValid() {
                        c.Set("X-Trace-ID", sc.TraceID().String())
                }

                // Record Prometheus metrics.
                duration := time.Since(startTime).Seconds()
                if cfg.HTTPMetrics != nil {
                        cfg.HTTPMetrics.RecordRequest(
                                c.Method(),
                                c.Route().Path,
                                strconv.Itoa(status),
                                duration,
                        )
                }

                return err
        }
}
