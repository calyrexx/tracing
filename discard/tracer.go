package discard

import (
	"context"
	"github.com/Calyr3x/tracing"
	"go.opentelemetry.io/otel/trace"
)

type tracer struct{}

func NewTracer() tracing.Tracer {
	return new(tracer)
}

func (t tracer) Start(ctx context.Context, name string) (context.Context, tracing.Span) {
	return ctx, new(span)
}

func (t tracer) TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}
