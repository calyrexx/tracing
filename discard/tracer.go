package discard

import (
	"context"

	"github.com/calyrexx/telemetry"
	"go.opentelemetry.io/otel/trace"
)

type tracer struct{}

func New() telemetry.Tracer {
	return new(tracer)
}

func (t tracer) Start(ctx context.Context, _ string) (context.Context, telemetry.Span) {
	return ctx, new(span)
}

func (t tracer) StartWith(
	ctx context.Context,
	_ string,
	_ ...trace.SpanStartOption,
) (context.Context, telemetry.Span) {
	return ctx, new(span)
}

func (t tracer) TraceIDFromContext(_ context.Context) string {
	return ""
}

func (t tracer) SpanFromContext(_ context.Context) telemetry.Span {
	return new(span)
}

func (t tracer) Metric() telemetry.Metric {
	return new(metric)
}
