package telemetry

import (
	"strings"

	"google.golang.org/grpc/metadata"
)

type MetadataCarrier struct {
	metadata.MD
}

func (c MetadataCarrier) Get(key string) string {
	vals := c.MD[strings.ToLower(key)]
	if len(vals) > 0 {
		return vals[0]
	}

	return ""
}

func (c MetadataCarrier) Set(key, val string) {
	k := strings.ToLower(key)
	c.MD[k] = []string{val}
}

func (c MetadataCarrier) Keys() []string {
	out := make([]string, 0, len(c.MD))
	for k := range c.MD {
		out = append(out, k)
	}

	return out
}
