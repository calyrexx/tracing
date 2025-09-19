package tracing

import (
	"context"
	"errors"
	"github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"log/slog"
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
	slogHandler     slog.Handler
}

// WithSlogHandler включает логирование через handler
func WithSlogHandler(handler slog.Handler) Option {
	return func(o *options) {
		o.slogHandler = handler
	}
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
	lp     *sdklog.LoggerProvider
	Logger *slog.Logger
}

// New создаёт tracer и возвращает его. Параметры serverName и endpoint являются обязательными.
func New(
	ctx context.Context,
	serverName string,
	endpoint string,
	opts ...Option,
) (*TracerWrapper, error) {
	if serverName == "" {
		return nil, errors.New("server name is required")
	}
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	o := &options{
		batchTimeout: time.Second * 5,
		sampler:      sdktrace.ParentBased(sdktrace.AlwaysSample()),
	}

	for _, fn := range opts {
		fn(o)
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

	// traces
	traceExpOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}

	if o.insecure {
		traceExpOpts = append(traceExpOpts, otlptracegrpc.WithInsecure())
	}

	traceExp, err := otlptracegrpc.New(ctx, traceExpOpts...)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp, sdktrace.WithBatchTimeout(o.batchTimeout)),
		sdktrace.WithSampler(o.sampler),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// logs
	var (
		lp     *sdklog.LoggerProvider
		logger *slog.Logger
	)

	logExpOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
	}

	if o.insecure {
		logExpOpts = append(logExpOpts, otlploggrpc.WithInsecure())
	}

	logExp, err := otlploggrpc.New(ctx, logExpOpts...)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, err
	}

	lp = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)

	otelSlog := otelslog.NewLogger(serverName, otelslog.WithLoggerProvider(lp))

	if o.slogHandler != nil {
		otelHandler := otelSlog.Handler()

		tee := slogmulti.Fanout(otelHandler, o.slogHandler)
		logger = slog.New(tee)
		slog.SetDefault(logger)
	}

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &TracerWrapper{
		tracer: tp.Tracer(serverName),
		tp:     tp,
		lp:     lp,
		Logger: logger,
	}, nil
}

// Shutdown останавливает провайдер трассировки
func (tw *TracerWrapper) Shutdown(ctx context.Context) error {
	var finalErr error

	if tw.tp != nil {
		if err := tw.tp.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			finalErr = err
		}
	}

	if tw.lp != nil {
		if err := tw.lp.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			finalErr = errors.Join(finalErr, err)
		}
	}

	return finalErr
}

// Start создаёт новый Span с указанным именем name и возвращает обновлённый контекст
// Каждый Span необходимо завершать.
func (tw *TracerWrapper) Start(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := tw.tracer.Start(ctx, name)

	return ctx, &spanWrapper{
		span: span,
	}
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
