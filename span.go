package tracing

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordError записывает ошибку в спан и устанавливает его статус как Error
func (j *Jaeger) RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// Start создаёт новый спан с указанным именем unit и возвращает обновлённый контекст
func (j *Jaeger) Start(ctx context.Context, unit string) (context.Context, trace.Span) {
	return j.tracer.Start(ctx, unit)
}

// SetAttributes устанавливает атрибут с именем unitName и значением, полученным из JSON-marshal req
func (j *Jaeger) SetAttributes(span trace.Span, req any, unitName string) {
	if reqJSON, mlErr := json.Marshal(req); mlErr == nil {
		span.SetAttributes(attribute.String(unitName, string(reqJSON)))
	}
}

// AddEvent добавляет событие с именем name и набором атрибутов attrs в спан
func (j *Jaeger) AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}
