package attribute

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

type KeyValue struct {
	Key   string
	Value any
}

func String(k string, v string) KeyValue {
	return KeyValue{
		Key:   k,
		Value: v,
	}
}

func Int(k string, v int) KeyValue {
	return KeyValue{
		Key:   k,
		Value: v,
	}
}

func Int64(k string, v int64) KeyValue {
	return KeyValue{
		Key:   k,
		Value: v,
	}
}

func Bool(k string, v bool) KeyValue {
	return KeyValue{
		Key:   k,
		Value: v,
	}
}

func Float64(k string, v float64) KeyValue {
	return KeyValue{
		Key:   k,
		Value: v,
	}
}

func ConvertAttrs(attrs []KeyValue) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	res := make([]attribute.KeyValue, 0, len(attrs))

	for _, a := range attrs {
		switch v := a.Value.(type) {
		case string:
			res = append(res, attribute.String(a.Key, v))
		case int:
			res = append(res, attribute.Int(a.Key, v))
		case int64:
			res = append(res, attribute.Int64(a.Key, v))
		case bool:
			res = append(res, attribute.Bool(a.Key, v))
		case float64:
			res = append(res, attribute.Float64(a.Key, v))
		default:
			res = append(res, attribute.String(a.Key, fmt.Sprintf("%v", v)))
		}
	}

	return res
}
