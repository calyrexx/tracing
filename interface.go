package tracing

import (
	"context"
)

type Span interface {
	End()
	SetStringAttribute(key, value string)
	SetIntAttribute(key string, value int)
	SetBoolAttribute(key string, value bool)
	SetJSONAttribute(key string, value interface{})
	AddEvent(name string)
	AddEventWithInt(name string, key string, value int)
	AddEventWithBool(name string, key string, value bool)
	AddEventWithString(name string, key string, value string)
	RecordError(err error)
}

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
	TraceIDFromContext(ctx context.Context) string
}
