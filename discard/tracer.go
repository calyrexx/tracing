package discard

import (
	"context"

	"github.com/calyrexx/telemetry"
)

type tracer struct{}

func New() telemetry.Tracer {
	return new(tracer)
}

func (t tracer) Start(ctx context.Context, name string) (context.Context, telemetry.Span) {
	return ctx, new(span)
}
