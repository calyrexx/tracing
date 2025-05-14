package tracing

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

// Config содержит настройки для подключения к Jaeger OTLP-экспортеру.
type Config struct {
	// HostName имя хоста, которое будет записано в ресурс трассировщика
	HostName string
	// ServiceName имя сервиса для создания спанов
	ServiceName string
	// Endpoint адрес OTLP/Jaeger collector (например, "localhost:4317")
	Endpoint string
}

type Jaeger struct {
	ServiceName    string
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
}

// New инициализирует провайдер трассировки OTLP (Jaeger) с указанными настройками
// и регистрирует глобальный TracerProvider и TextMapPropagator.
func New(ctx context.Context, cfg Config) (*Jaeger, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.HostNameKey.String(cfg.HostName),
		)),
	)
	// Регистрируем глобальный TracerProvider
	otel.SetTracerProvider(tp)

	// Настраиваем пропагатор W3C TraceContext + Baggage
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &Jaeger{
		ServiceName:    cfg.ServiceName,
		tracer:         tp.Tracer(cfg.ServiceName),
		tracerProvider: tp,
	}, nil
}

// Shutdown завершает работу TracerProvider и дожидается отправки всех спанов
func (j *Jaeger) Shutdown(ctx context.Context) error {
	return j.tracerProvider.Shutdown(ctx)
}
