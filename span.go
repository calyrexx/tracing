package tracing

import (
	"bytes"
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"sync"
)

// RecordError записывает ошибку в спан и устанавливает его статус как Error
func (tw *TracerWrapper) RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// RecordErrorWithDetails записывает ошибку с дополнительными деталями
func (tw *TracerWrapper) RecordErrorWithDetails(span trace.Span, err error, details map[string]interface{}) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	var attrs []attribute.KeyValue
	for k, v := range details {
		attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", v)))
	}
	span.SetAttributes(attrs...)
}

// Start создаёт новый спан с указанным именем unit и возвращает обновлённый контекст
func (tw *TracerWrapper) Start(ctx context.Context, unit string) (context.Context, trace.Span) {
	return tw.tracer.Start(ctx, unit)
}

// SetStringAttribute устанавливает строковый атрибут
func (tw *TracerWrapper) SetStringAttribute(span trace.Span, key, value string) {
	span.SetAttributes(attribute.String(key, value))
}

// SetIntAttribute устанавливает числовой атрибут
func (tw *TracerWrapper) SetIntAttribute(span trace.Span, key string, value int) {
	span.SetAttributes(attribute.Int(key, value))
}

// SetBoolAttribute устанавливает булевый атрибут
func (tw *TracerWrapper) SetBoolAttribute(span trace.Span, key string, value bool) {
	span.SetAttributes(attribute.Bool(key, value))
}

// SetJSONAttribute сериализует объект в JSON и устанавливает как атрибут
func (tw *TracerWrapper) SetJSONAttribute(span trace.Span, key string, value interface{}) {
	buf := jsonBufferPool.Get().(*bytes.Buffer)
	defer jsonBufferPool.Put(buf)
	buf.Reset()

	if err := json.NewEncoder(buf).Encode(value); err == nil {
		span.SetAttributes(attribute.String(key, buf.String()))
	}
}

var jsonBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// AddEvent добавляет событие с именем name и набором атрибутов attrs в спан
func (tw *TracerWrapper) AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// TraceIDFromContext возвращает текущий TraceID из контекста или пустую строку, если спан не валиден
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}
