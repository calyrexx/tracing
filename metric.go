package telemetry

import (
	"context"
	"fmt"
	"sync"

	"github.com/calyrexx/telemetry/attribute"
	"go.opentelemetry.io/otel/metric"
)

type (
	Metric interface {
		Counter(name string) (Counter, error)
		Gauge(name string) (Gauge, error)
		Histogram(name string) (Histogram, error)
	}

	Counter interface {
		Add(ctx context.Context, value float64, attrs ...attribute.KeyValue)
	}

	Gauge interface {
		Add(ctx context.Context, value float64, attrs ...attribute.KeyValue)
	}

	Histogram interface {
		Record(ctx context.Context, value float64, attrs ...attribute.KeyValue)
	}
)

type metricWrapper struct {
	meter      metric.Meter
	counters   sync.Map // map[string]Counter
	gauges     sync.Map // map[string]Gauge
	histograms sync.Map // map[string]Histogram
}

func (tw *Wrapper) Metric() Metric {
	return &metricWrapper{
		meter: tw.meter,
	}
}

func (m *metricWrapper) Counter(name string) (Counter, error) {
	if m.meter == nil {
		return nil, fmt.Errorf("meter is nil")
	}

	if c, ok := m.counters.Load(name); ok {
		if res, ok2 := c.(Counter); ok2 {
			return res, nil
		} else {
			return nil, fmt.Errorf("counter is not Counter")
		}
	}

	otelCounter, err := m.meter.Float64Counter(name)
	if err != nil {
		return nil, fmt.Errorf("error creating counter: %w", err)
	}

	w := &counterWrapper{
		c: otelCounter,
	}
	m.counters.Store(name, w)

	return w, nil
}

func (m *metricWrapper) Gauge(name string) (Gauge, error) {
	if m.meter == nil {
		return nil, fmt.Errorf("meter is nil")
	}

	if g, ok := m.gauges.Load(name); ok {
		if res, ok2 := g.(Gauge); ok2 {
			return res, nil
		} else {
			return nil, fmt.Errorf("gauge is not Gauge")
		}
	}

	otelGauge, err := m.meter.Float64UpDownCounter(name)
	if err != nil {
		return nil, fmt.Errorf("error creating gauge: %w", err)
	}

	w := &gaugeWrapper{
		g: otelGauge,
	}
	m.gauges.Store(name, w)

	return w, nil
}

func (m *metricWrapper) Histogram(name string) (Histogram, error) {
	if m.meter == nil {
		return nil, fmt.Errorf("meter is nil")
	}

	if h, ok := m.histograms.Load(name); ok {
		if res, ok2 := h.(Histogram); ok2 {
			return res, nil
		} else {
			return nil, fmt.Errorf("histogram is not Histogram")
		}
	}

	otelHist, err := m.meter.Float64Histogram(name)
	if err != nil {
		return nil, fmt.Errorf("error creating histogram: %w", err)
	}

	w := &histogramWrapper{
		h: otelHist,
	}
	m.histograms.Store(name, w)

	return w, nil
}

type counterWrapper struct {
	c metric.Float64Counter
}

func (c *counterWrapper) Add(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	if c == nil {
		return
	}

	c.c.Add(ctx, value, metric.WithAttributes(attribute.ConvertAttrs(attrs)...))
}

type gaugeWrapper struct {
	g metric.Float64UpDownCounter
}

func (g *gaugeWrapper) Add(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	if g == nil {
		return
	}

	g.g.Add(ctx, value, metric.WithAttributes(attribute.ConvertAttrs(attrs)...))
}

type histogramWrapper struct {
	h metric.Float64Histogram
}

func (h *histogramWrapper) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	if h == nil {
		return
	}

	h.h.Record(ctx, value, metric.WithAttributes(attribute.ConvertAttrs(attrs)...))
}
