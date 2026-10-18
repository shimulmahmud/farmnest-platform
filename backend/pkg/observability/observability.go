package observability

import (
    "context"
    "fmt"
    "log"
    "net/http"

    promclient "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.opentelemetry.io/otel"
    promexporter "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/propagation"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
    "go.opentelemetry.io/otel/trace"
)

type Provider struct {
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
    shutdown       func(context.Context) error
}

func NewProvider(serviceName string, prometheusPort int) (*Provider, error) {
    res, err := resource.Merge(
        resource.Default(),
        resource.NewSchemaless(
            semconv.ServiceNameKey.String(serviceName),
        ),
    )
    if err != nil {
        return nil, err
    }

    registry := promclient.NewRegistry()

    promExporter, err := promexporter.New(promexporter.WithRegisterer(registry))
    if err != nil {
        return nil, err
    }

    meterProvider := sdkmetric.NewMeterProvider(
        sdkmetric.WithResource(res),
        sdkmetric.WithReader(promExporter),
    )

    traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
    if err != nil {
        return nil, err
    }

    tracerProvider := sdktrace.NewTracerProvider(
        sdktrace.WithResource(res),
        sdktrace.WithBatcher(traceExporter),
    )

    otel.SetTracerProvider(tracerProvider)
    otel.SetMeterProvider(meterProvider)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    go serveMetrics(registry, prometheusPort)

    log.Printf("Observability initialized for service %q, Prometheus on :%d/metrics", serviceName, prometheusPort)

    return &Provider{
        TracerProvider: tracerProvider,
        MeterProvider:  meterProvider,
        shutdown: func(ctx context.Context) error {
            var errs []error
            if err := tracerProvider.Shutdown(ctx); err != nil {
                errs = append(errs, err)
            }
            if err := meterProvider.Shutdown(ctx); err != nil {
                errs = append(errs, err)
            }
            if len(errs) > 0 {
                return errs[0]
            }
            return nil
        },
    }, nil
}

func serveMetrics(registry *promclient.Registry, port int) {
    mux := http.NewServeMux()
    handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
    mux.Handle("/metrics", handler)
    addr := fmt.Sprintf(":%d", port)
    log.Printf("Prometheus metrics server listening on %s/metrics", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Printf("Prometheus metrics server error: %v", err)
    }
}

func (p *Provider) Shutdown(ctx context.Context) error {
    return p.shutdown(ctx)
}

func (p *Provider) Tracer(name string) trace.Tracer {
    return p.TracerProvider.Tracer(name)
}

func (p *Provider) Meter(name string) metric.Meter {
    return p.MeterProvider.Meter(name)
}