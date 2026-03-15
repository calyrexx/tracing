package discard

import (
	"context"

	"github.com/calyrexx/telemetry"
	"github.com/calyrexx/telemetry/attribute"
)

type (
	metric struct{}

	counter struct{}

	gauge struct{}

	histogram struct{}
)

func (m metric) Counter(name string) (telemetry.Counter, error) {
	return new(counter), nil
}

func (m metric) Gauge(name string) (telemetry.Gauge, error) {
	return new(gauge), nil
}

func (m metric) Histogram(name string) (telemetry.Histogram, error) {
	return new(histogram), nil
}

func (c counter) Add(ctx context.Context, value float64, attrs ...attribute.KeyValue) {}

func (c gauge) Add(ctx context.Context, value float64, attrs ...attribute.KeyValue) {}

func (c histogram) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {}
