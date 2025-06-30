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

// WithServiceVersion задаёт версию в ресурсах трассировки
func WithServiceVersion(version string) Option {
	return func(o *options) {
		o.extraAttributes = append(o.extraAttributes,
			semconv.ServiceVersionKey.String(version),
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
	tracer trace.Tracer
	tp     *sdktrace.TracerProvider
}

// New создаёт tracer и возвращает его. Параметры serverName и endpoint являются обязательными.
func New(
	ctx context.Context,
	serverName string,
	endpoint string,
	opts ...Option,
) (*TracerWrapper, error) {
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
		return nil, err
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
		return nil, err
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
		tracer: tp.Tracer(serverName),
		tp:     tp,
	}, nil
}

// Shutdown останавливает провайдер трассировки
func (tw *TracerWrapper) Shutdown(ctx context.Context) error {
	if tw.tp == nil {
		return nil
	}
	if err := tw.tp.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// Start создаёт новый Span с указанным именем name и возвращает обновлённый контекст
// Каждый Span необходимо завершать.
func (tw *TracerWrapper) Start(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := tw.tracer.Start(ctx, name)
	return ctx, &spanWrapper{span: span}
}

// TraceIDFromContext возвращает текущий TraceID из контекста или пустую строку, если спан не валиден
func (tw *TracerWrapper) TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

func NewSubTracer(name string) *TracerWrapper {
	return &TracerWrapper{
		tracer: otel.GetTracerProvider().Tracer(name),
	}
}
