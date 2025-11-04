package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

type Wrapper struct {
	tracer trace.Tracer
	tp     *sdktrace.TracerProvider
	lp     *sdklog.LoggerProvider
}

// New создаёт Wrapper и возвращает его. Параметры serverName и endpoint являются обязательными.
func New(
	ctx context.Context,
	serverName string,
	endpoint string,
	options ...Option,
) (*Wrapper, error) {
	if serverName == "" {
		return nil, errors.New("server name is required")
	}

	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	opts := &telemetryOptions{
		tracer: tracerOptions{
			batchTimeout: 5 * time.Second,
			sampler:      sdktrace.ParentBased(sdktrace.AlwaysSample()),
		},
		logger: loggerOptions{
			level: slog.LevelInfo,
		},
	}

	for _, opt := range options {
		opt(opts)
	}

	res, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceNameKey.String(serverName)),
		resource.WithAttributes(opts.resAttrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var tp *sdktrace.TracerProvider
	if opts.enableTraces {
		tp, err = newTracerProvider(ctx, endpoint, opts, res)
		if err != nil {
			return nil, err
		}
	}

	var lp *sdklog.LoggerProvider
	if opts.enableLogs {
		lp, err = newLogsProvider(ctx, endpoint, serverName, opts, res)
		if err != nil {
			return nil, err
		}
	}

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	var tr trace.Tracer
	if tp != nil {
		tr = tp.Tracer(serverName)
	} else {
		tr = otel.GetTracerProvider().Tracer(serverName)
	}

	return &Wrapper{
		tracer: tr,
		tp:     tp,
		lp:     lp,
	}, nil
}

func newLogsProvider(
	ctx context.Context,
	endpoint string,
	serverName string,
	opts *telemetryOptions,
	res *resource.Resource,
) (*sdklog.LoggerProvider, error) {
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

	var proc sdklog.Processor = sdklog.NewBatchProcessor(
		logExp,
		sdklog.WithExportInterval(opts.logger.batchTimeout),
	)

	proc = minsev.NewLogProcessor(proc, minsev.Severity(opts.logger.level))

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(proc),
		sdklog.WithResource(res),
	)

	otelHandler := otelslog.NewLogger(serverName, otelslog.WithLoggerProvider(lp)).Handler()

	var h = otelHandler

	if opts.logger.slogHandler != nil {
		h = slogmulti.Fanout(otelHandler, opts.logger.slogHandler)
	}

	slog.SetDefault(slog.New(h))

	return lp, nil
}

func newTracerProvider(
	ctx context.Context,
	endpoint string,
	opts *telemetryOptions,
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
		sdktrace.WithBatcher(traceExp, sdktrace.WithBatchTimeout(opts.tracer.batchTimeout)),
		sdktrace.WithSampler(opts.tracer.sampler),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp, nil
}

// Shutdown останавливает провайдер телеметрии.
func (tw *Wrapper) Shutdown(ctx context.Context) error {
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
//
//	ctx, span := tracer.Start(ctx, "example.Method")
//	defer span.End()
func (tw *Wrapper) Start(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := tw.tracer.Start(ctx, name)

	return ctx, &spanWrapper{
		span: span,
	}
}

// TraceIDFromContext возвращает текущий TraceID из контекста или пустую строку, если спан не валиден.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()

	if sc.IsValid() {
		return sc.TraceID().String()
	}

	return ""
}

func SpanFromContext(ctx context.Context) Span {
	return &spanWrapper{
		span: trace.SpanFromContext(ctx),
	}
}

func NewSubTracer(name string) *Wrapper {
	return &Wrapper{
		tracer: otel.GetTracerProvider().Tracer(name),
	}
}
