package telemetry

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type Option func(*telemetryOptions)

type TracerOption func(*tracerOptions)

type LoggerOption func(*loggerOptions)

type telemetryOptions struct {
	insecure     bool
	resAttrs     []attribute.KeyValue
	enableTraces bool
	enableLogs   bool
	tracer       tracerOptions
	logger       loggerOptions
}

type tracerOptions struct {
	batchTimeout time.Duration
	sampler      sdktrace.Sampler
}

type loggerOptions struct {
	batchTimeout time.Duration
	level        slog.Level
	slogHandler  slog.Handler
}

func WithInsecure() Option {
	return func(c *telemetryOptions) {
		c.insecure = true
	}
}

func WithResourceAttribute(a attribute.KeyValue) Option {
	return func(c *telemetryOptions) {
		c.resAttrs = append(c.resAttrs, a)
	}
}

func WithEnvironment(env string) Option {
	return WithResourceAttribute(attribute.String("deployment.environment", env))
}

func WithHostName(h string) Option {
	return WithResourceAttribute(semconv.HostNameKey.String(h))
}

func WithServiceVersion(v string) Option {
	return WithResourceAttribute(semconv.ServiceVersionKey.String(v))
}

func WithTracer(opts ...TracerOption) Option {
	return func(c *telemetryOptions) {
		c.enableTraces = true
		for _, o := range opts {
			o(&c.tracer)
		}
	}
}

func WithSampler(s sdktrace.Sampler) TracerOption {
	return func(t *tracerOptions) {
		t.sampler = s
	}
}

func WithTraceBatchTimeout(d time.Duration) TracerOption {
	return func(t *tracerOptions) {
		t.batchTimeout = d
	}
}

func WithLogger(opts ...LoggerOption) Option {
	return func(c *telemetryOptions) {
		c.enableLogs = true
		for _, o := range opts {
			o(&c.logger)
		}
	}
}

func WithLogLevel(lvl slog.Level) LoggerOption {
	return func(l *loggerOptions) {
		l.level = lvl
	}
}

func WithLogBatchTimeout(d time.Duration) LoggerOption {
	return func(l *loggerOptions) {
		l.batchTimeout = d
	}
}

func WithSlogHandler(h slog.Handler) LoggerOption {
	return func(l *loggerOptions) {
		l.slogHandler = h
	}
}
