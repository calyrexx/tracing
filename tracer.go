package tracing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

type Option func(*tracerOptions)

// tracerOptions внутренний конфиг по-умолчанию.
type tracerOptions struct {
	insecure        bool
	batchTimeout    time.Duration
	sampler         sdktrace.Sampler
	extraAttributes []attribute.KeyValue
	slogHandler     slog.Handler
}

// WithSlogHandler включает логирование через handler.
func WithSlogHandler(handler slog.Handler) Option {
	return func(o *tracerOptions) {
		o.slogHandler = handler
	}
}

// WithInsecure отключает TLS.
func WithInsecure() Option {
	return func(o *tracerOptions) {
		o.insecure = true
	}
}

// WithHostName задаёт атрибут хоста в ресурсах.
func WithHostName(host string) Option {
	return func(o *tracerOptions) {
		o.extraAttributes = append(o.extraAttributes,
			semconv.HostNameKey.String(host),
		)
	}
}

// WithEnvironment задаёт зону (prod/dev/stage) в ресурсах.
func WithEnvironment(env string) Option {
	return func(o *tracerOptions) {
		o.extraAttributes = append(o.extraAttributes,
			attribute.String("deployment.environment", env),
		)
	}
}

// WithServiceVersion задаёт версию в ресурсах.
func WithServiceVersion(version string) Option {
	return func(o *tracerOptions) {
		o.extraAttributes = append(o.extraAttributes,
			semconv.ServiceVersionKey.String(version),
		)
	}
}

// WithBatchTimeout задаёт максимальное время буферизации.
func WithBatchTimeout(d time.Duration) Option {
	return func(o *tracerOptions) {
		o.batchTimeout = d
	}
}

// WithSampler позволяет задать стратегию семплинга.
func WithSampler(s sdktrace.Sampler) Option {
	return func(o *tracerOptions) {
		o.sampler = s
	}
}

// WithResourceAttribute добавляет произвольный атрибут к ресурсам.
func WithResourceAttribute(attr attribute.KeyValue) Option {
	return func(o *tracerOptions) {
		o.extraAttributes = append(o.extraAttributes, attr)
	}
}

type TracerWrapper struct {
	tracer trace.Tracer
	tp     *sdktrace.TracerProvider
	lp     *sdklog.LoggerProvider
}

// New создаёт tracer и возвращает его. Параметры serverName и endpoint являются обязательными.
func New(
	ctx context.Context,
	serverName string,
	endpoint string,
	options ...Option,
) (*TracerWrapper, error) {
	if serverName == "" {
		return nil, errors.New("server name is required")
	}

	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	opts := &tracerOptions{
		batchTimeout: time.Second * 5,
		sampler:      sdktrace.ParentBased(sdktrace.AlwaysSample()),
	}

	for _, fn := range options {
		fn(opts)
	}

	res, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serverName),
			semconv.HostNameKey.String(serverName),
		),
		resource.WithAttributes(opts.extraAttributes...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp, err := newTracerProvider(ctx, endpoint, opts, res)
	if err != nil {
		return nil, err
	}

	lp, err := newLogsProvider(ctx, endpoint, serverName, opts, res)
	if err != nil {
		return nil, err
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
	}, nil
}

func newLogsProvider(
	ctx context.Context,
	endpoint string,
	serverName string,
	opts *tracerOptions,
	res *resource.Resource) (*sdklog.LoggerProvider, error) {
	var (
		lp     *sdklog.LoggerProvider
		logger *slog.Logger
	)

	logExpOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
	}

	if opts.insecure {
		logExpOpts = append(logExpOpts, otlploggrpc.WithInsecure())
	}

	logExp, err := otlploggrpc.New(ctx, logExpOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)

	otelSlog := otelslog.NewLogger(serverName, otelslog.WithLoggerProvider(lp))

	if opts.slogHandler != nil {
		otelHandler := otelSlog.Handler()

		tee := slogmulti.Fanout(otelHandler, opts.slogHandler)
		logger = slog.New(tee)
		slog.SetDefault(logger)
	}

	return lp, nil
}

func newTracerProvider(
	ctx context.Context,
	endpoint string,
	opts *tracerOptions,
	res *resource.Resource,
) (*sdktrace.TracerProvider, error) {
	traceExpOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}

	if opts.insecure {
		traceExpOpts = append(traceExpOpts, otlptracegrpc.WithInsecure())
	}

	traceExp, err := otlptracegrpc.New(ctx, traceExpOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp, sdktrace.WithBatchTimeout(opts.batchTimeout)),
		sdktrace.WithSampler(opts.sampler),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp, nil
}

// Shutdown останавливает провайдер трассировки.
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

// Start создаёт новый Span с указанным именем name и возвращает обновлённый контекст.
// Каждый Span необходимо завершать.
func (tw *TracerWrapper) Start(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := tw.tracer.Start(ctx, name)

	return ctx, &spanWrapper{
		span: span,
	}
}

// TraceIDFromContext возвращает текущий TraceID из контекста или пустую строку, если спан не валиден.
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
