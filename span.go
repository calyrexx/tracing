package telemetry

import (
	"fmt"

	"github.com/bytedance/sonic"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Span interface {
	End()
	SetStatus(code codes.Code, description string)
	SetStringAttribute(key, value string)
	SetIntAttribute(key string, value int)
	SetInt64Attribute(key string, value int64)
	SetBoolAttribute(key string, value bool)
	SetJSONAttribute(key string, value any)
	AddEvent(name string)
	AddEventWithInt(name string, key string, value int)
	AddEventWithBool(name string, key string, value bool)
	AddEventWithString(name string, key string, value string)
	RecordError(err error)
}

type spanWrapper struct {
	span trace.Span
}

// End корректно завершает span.
func (s *spanWrapper) End() {
	s.span.End()
}

func (s *spanWrapper) SetStatus(code codes.Code, description string) {
	s.span.SetStatus(code, description)
}

// SetStringAttribute устанавливает строковый атрибут.
func (s *spanWrapper) SetStringAttribute(key, value string) {
	s.span.SetAttributes(attribute.String(key, value))
}

// SetIntAttribute устанавливает числовой атрибут.
func (s *spanWrapper) SetIntAttribute(key string, value int) {
	s.span.SetAttributes(attribute.Int(key, value))
}

// SetInt64Attribute устанавливает int64 атрибут.
func (s *spanWrapper) SetInt64Attribute(key string, value int64) {
	s.span.SetAttributes(attribute.Int64(key, value))
}

// SetBoolAttribute устанавливает булевый атрибут.
func (s *spanWrapper) SetBoolAttribute(key string, value bool) {
	s.span.SetAttributes(attribute.Bool(key, value))
}

// SetJSONAttribute сериализует объект в JSON и устанавливает как атрибут.
func (s *spanWrapper) SetJSONAttribute(key string, value any) {
	if str, err := sonic.MarshalString(value); err == nil {
		s.span.SetAttributes(attribute.String(key, str))
	}
}

// AddEvent добавляет событие с именем name в спан.
func (s *spanWrapper) AddEvent(name string) {
	s.span.AddEvent(name)
}

// AddEventWithInt добавляет событие с именем name и значением key (string) value (int) в спан.
func (s *spanWrapper) AddEventWithInt(name string, key string, value int) {
	s.span.AddEvent(name, trace.WithAttributes(attribute.Int(key, value)))
}

// AddEventWithBool добавляет событие с именем name и значением key (string) value (bool) в спан.
func (s *spanWrapper) AddEventWithBool(name string, key string, value bool) {
	s.span.AddEvent(name, trace.WithAttributes(attribute.Bool(key, value)))
}

// AddEventWithString добавляет событие с именем name и значением key (string) value (string) в спан.
func (s *spanWrapper) AddEventWithString(name string, key string, value string) {
	s.span.AddEvent(name, trace.WithAttributes(attribute.String(key, value)))
}

// RecordError записывает ошибку в спан и устанавливает его статус как Error.
func (s *spanWrapper) RecordError(err error) {
	s.span.RecordError(err)
}

// RecordErrorWithDetails записывает ошибку с дополнительными деталями.
func (s *spanWrapper) RecordErrorWithDetails(err error, details map[string]any) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())

	attrs := make([]attribute.KeyValue, 0, len(details))
	for k, v := range details {
		attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", v)))
	}

	s.span.SetAttributes(attrs...)
}
