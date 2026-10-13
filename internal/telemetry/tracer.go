package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitTracer(ctx context.Context, serviceName, collectorAddr string) (trace.Tracer, func(context.Context) error, error) {
	if collectorAddr == "" {
		collectorAddr = "otel-collector:4317"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao criar resource otel: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctxTimeout,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(collectorAddr),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao criar exporter otlp grpc: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tracerProvider.Tracer(serviceName)

	shutdown := func(shutdownCtx context.Context) error {
		return tracerProvider.Shutdown(shutdownCtx)
	}

	return tracer, shutdown, nil
}
