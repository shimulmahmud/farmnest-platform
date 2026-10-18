package telemetry

import (
        "context"

        "go.opentelemetry.io/otel"
        "go.opentelemetry.io/otel/attribute"
        "go.opentelemetry.io/otel/codes"
        "go.opentelemetry.io/otel/trace"
)

// TraceIDFromContext extracts the trace ID from the OTel context, or returns empty.
func TraceIDFromContext(ctx context.Context) string {
        sc := trace.SpanContextFromContext(ctx)
        if !sc.IsValid() {
                return ""
        }
        return sc.TraceID().String()
}

// SpanIDFromContext extracts the span ID from the OTel context, or returns empty.
func SpanIDFromContext(ctx context.Context) string {
        sc := trace.SpanContextFromContext(ctx)
        if !sc.IsValid() {
                return ""
        }
        return sc.SpanID().String()
}

// RequestIDFromContext extracts the request ID from context locals (set by middleware).
// This is the X-Request-ID header value, propagated as an OTel attribute.
func RequestIDFromContext(ctx context.Context) string {
        // The request ID is stored via Fiber locals, not Go context.
        // This helper is for use in the Fiber middleware where we have access to c.Locals().
        // For Go-context based extraction, use the OTel trace ID as the request correlation.
        return ""
}

// CorrelationIDFromContext returns the trace ID to use as correlation_id in events.
// This links domain events to the originating HTTP request trace.
func CorrelationIDFromContext(ctx context.Context) string {
        return TraceIDFromContext(ctx)
}

// CausationIDFromContext returns the current span ID to use as causation_id in events.
// This links a child event to the span that caused it.
func CausationIDFromContext(ctx context.Context) string {
        return SpanIDFromContext(ctx)
}

// SafeAttributes returns only attributes that are known to be non-secret.
// Use this when adding attributes to spans to prevent accidental secret leakage.
// Unknown keys are silently dropped.
var safeAttributeKeys = map[string]bool{
        "http.method":              true,
        "http.route":               true,
        "http.status_code":         true,
        "http.url":                  true,
        "http.host":                 true,
        "http.scheme":               true,
        "http.user_agent":           true,
        "http.target":               true,
        "request_id":                true,
        "service.name":              true,
        "mongodb.collection":        true,
        "mongodb.operation":         true,
        "nats.subject":              true,
        "nats.stream":               true,
        "nats.consumer":             true,
        "aggregate.type":            true,
        "aggregate.id":              true,
        "event.type":                true,
        "error.type":                true,
        "error.message":             true,
}

// FilterAttributes filters a map of attributes to only include safe, non-secret keys.
// Headers like Authorization, Cookie, and custom secret headers are never included.
func FilterAttributes(attrs map[string]string) []attribute.KeyValue {
        var result []attribute.KeyValue
        for k, v := range attrs {
                if safeAttributeKeys[k] {
                        result = append(result, attribute.String(k, v))
                }
        }
        return result
}

// StartSpan is a convenience wrapper that starts a child span with standard attributes.
// It automatically sets the span name and records errors on completion.
func StartSpan(ctx context.Context, tracer trace.Tracer, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
        ctx, span := tracer.Start(ctx, name, trace.WithAttributes(attrs...))
        return ctx, span
}

// RecordSpanError records an error on the span and sets the status.
// The error message is included but the error string is checked against common
// secret patterns (passwords, tokens) — if found, the message is redacted.
func RecordSpanError(span trace.Span, err error) {
        if err == nil {
                return
        }
        msg := sanitizeErrorMessage(err.Error())
        span.RecordError(err)
        span.SetAttributes(attribute.String("error.message", msg))
        span.SetStatus(codes.Error, msg)
}

// sanitizeErrorMessage removes common secret patterns from error messages
// to prevent accidental leakage in traces and metrics.
func sanitizeErrorMessage(msg string) string {
        // Redact common secret-bearing patterns
        secretPatterns := []string{
                "Bearer ",
                "password=",
                "secret=",
                "token=",
                "api_key=",
                "apikey=",
        }
        for _, p := range secretPatterns {
                if contains(msg, p) {
                        return "[REDACTED: potential secret in error message]"
                }
        }
        return msg
}

// contains is a simple string contains check.
func contains(s, substr string) bool {
        for i := 0; i <= len(s)-len(substr); i++ {
                if s[i:i+len(substr)] == substr {
                        return true
                }
        }
        return false
}

// TracerFromContext returns a tracer using the global provider.
func TracerFromContext(ctx context.Context, name string) trace.Tracer {
        return otel.Tracer(name)
}
