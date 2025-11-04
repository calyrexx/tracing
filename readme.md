# telemetry 🌐

Минималистичная обёртка над OpenTelemetry для **трейсов** и **логов** (slog).

## Установка

```bash
go get github.com/calyrexx/telemetry
```

## Quick start

```go
tw, err := telemetry.New(ctx,
  "test-app",                 // service.name
  "otel-collector:4317",      // OTLP gRPC endpoint
  telemetry.WithInsecure(),   // если без TLS
  telemetry.WithServiceVersion("v1.0.0"),
  telemetry.WithEnvironment("local"),
  telemetry.WithTracer(       // включить трейсы
    telemetry.WithTraceBatchTimeout(2*time.Second),
  ),
  telemetry.WithLogger(       // включить логи → slog
    telemetry.WithLogLevel(slog.LevelInfo),
  ),
)
if err != nil { log.Fatal(err) }
defer tw.Shutdown(ctx)
```

> Пакет сам настраивает `propagation`, `resource` (`service.name`, версия и пр.) и устанавливает `slog.SetDefault(...)` c OTLP-хендлером (можно сделать фан-аунт своего хендлера через `WithSlogHandler`).

## Трейсы: создание спанов

```go
ctx, span := tw.Start(ctx, "usecase.Process")
defer span.End()

span.SetStringAttribute("key", "value")
span.AddEvent("fetch_started")
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

Вытянуть `trace_id` из контекста:

```go
tid := telemetry.TraceIDFromContext(ctx)
```

Саб-трейсер для подсистемы:

```go
dbTracer := telemetry.NewSubTracer("db")
ctx, s := dbTracer.Start(ctx, "db.Query")
defer s.End()
```

## Логи в OTLP (через slog)

```go
slog.Info("user created", "id", id)  // полетит в OTLP (минимальный уровень — через WithLogLevel)
```

## gRPC

**Server:**

```go
s := grpc.NewServer(
  grpc.StatsHandler(telemetry.StatsServerHandler()),
  grpc.UnaryInterceptor(telemetry.UnaryTracingInterceptor()),
)
```

**Client:**

```go
conn, _ := grpc.Dial(
  addr,
  grpc.WithInsecure(),
  grpc.WithStatsHandler(telemetry.StatsClientHandler()),
  grpc.WithUnaryInterceptor(telemetry.UnaryPropagationInterceptor()),
)
```

`UnaryTracingInterceptor` пишет `rpc.request/response` (JSON) в текущий спан и проставляет статус.

## Завершение

```go
_ = tw.Shutdown(ctx) // корректно закрывает trace и log провайдеры
```

---

**Назначение пакета:** быстро включить OTLP-трейсинг и OTLP-логирование с минимальной конфигурацией.

Для продакшена используйте TLS (уберите WithInsecure()).
