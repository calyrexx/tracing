package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

// Option — функция, настраивающая поведение New
type Option func(*options)

// внутренний конфиг по-умолчанию
type options struct {
	insecure        bool
	batchTimeout    time.Duration
	sampler         sdktrace.Sampler
	extraAttributes []attribute.KeyValue
}

// WithInsecure отключает TLS
func WithInsecure() Option {
	return func(o *options) {
		o.insecure = true
	}
}

// WithHostName задаёт атрибут хоста в ресурсах трассировки
func WithHostName(host string) Option {
    return func(o *options) {
        o.extraAttributes = append(o.extraAttributes,
            semconv.HostNameKey.String(host),
        )
    }
}

// WithEnvironment задаёт зону (prod/dev/stage) в ресурсах трассировки
func WithEnvironment(env string) Option {
    return func(o *options) {
        o.extraAttributes = append(o.extraAttributes,
            attribute.String("deployment.environment", env),
        )
    }
}

// WithBatchTimeout задаёт максимальное время буферизации
func WithBatchTimeout(d time.Duration) Option {
	return func(o *options) {
		o.batchTimeout = d
	}
}

// WithSampler позволяет задать стратегию семплинга
func WithSampler(s sdktrace.Sampler) Option {
	return func(o *options) {
		o.sampler = s
	}
}

// WithResourceAttribute добавляет произвольный атрибут к ресурсам
func WithResourceAttribute(attr attribute.KeyValue) Option {
	return func(o *options) {
		o.extraAttributes = append(o.extraAttributes, attr)
	}
}

type TracerWrapper struct {
	Tracer   trace.Tracer
}

type ShutdownFunc func(ctx context.Context) error

// New создаёт tracer и возвращает его вместе с функцией Shutdown.
// Параметры serverName и endpoint являются обязательными.
func New(
	ctx context.Context,
	serverName string,
	endpoint string,
	opts ...Option,
) (*TracerWrapper, ShutdownFunc, error) {
	o := &options{
		batchTimeout: time.Second * 5,
		sampler:      sdktrace.ParentBased(sdktrace.AlwaysSample()),
	}
	for _, fn := range opts {
		fn(o)
	}

	expOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if o.insecure {
		expOpts = append(expOpts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptracegrpc.New(ctx, expOpts...)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serverName),
			semconv.HostNameKey.String(serverName),
		),
		resource.WithAttributes(o.extraAttributes...),
	)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(o.batchTimeout)),
		sdktrace.WithSampler(o.sampler),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &TracerWrapper{
		Tracer: tp.Tracer(serverName),
	}, tp.Shutdown, nil
}
