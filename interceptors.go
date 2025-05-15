package tracing

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type metadataCarrier struct {
	metadata.MD
}

func (c metadataCarrier) Get(key string) string {
	vals := c.MD[strings.ToLower(key)]
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (c metadataCarrier) Set(key, val string) {
	k := strings.ToLower(key)
	c.MD[k] = []string{val}
}

func (c metadataCarrier) Keys() []string {
	out := make([]string, 0, len(c.MD))
	for k := range c.MD {
		out = append(out, k)
	}
	return out
}

// PropagationUnaryInterceptor распространяет трейс через метаданные gRPC
func PropagationUnaryInterceptor() grpc.UnaryClientInterceptor {
	propagator := otel.GetTextMapPropagator()
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		carrier := metadataCarrier{MD: md}
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, carrier.MD)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// TracingUnaryInterceptor добавляет трейсинг для входящих gRPC-запросов
func TracingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		span := trace.SpanFromContext(ctx)

		span.SetAttributes(attribute.String("rpc.method", info.FullMethod))

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-request-id"); len(ids) > 0 {
				span.SetAttributes(attribute.String("x-request-id", ids[0]))
			}
		}

		if reqJSON, err := json.Marshal(req); err == nil {
			request := string(reqJSON)
			span.SetAttributes(attribute.String("rpc.request", request))
		}

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}
