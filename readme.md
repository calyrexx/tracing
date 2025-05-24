# tracing 🌐

Лёгкая обёртка для OpenTelemetry SDK на Go: инициализация трейсинга, удобная работа со спанами, интеграция с gRPC, расширяемый API и чистые интерфейсы для проектирования микросервисов.

---

## Возможности

* Быстрая инициализация трейсера (`tracing.New`) с опциями (`insecure`, версии, окружение и др.)
* Гибкое управление ресурсными атрибутами трейсера
* Удобный API для работы со спанами: атрибуты, события, сериализация объектов в JSON-атрибуты, ошибки, завершение
* Интерфейсы Tracer/Span для чистой архитектуры и моков
* Готовые gRPC-интерсепторы: прозрачная поддержка propagation (TraceContext) и автоматическая запись метаданных/ошибок

---

## Быстрый старт

```bash
go get github.com/calyrexx/tracing
```

```go
import (
    "context"
    "log"
    "github.com/Calyr3x/tracing"
)

func main() {
    ctx := context.Background()
    tracer, err := tracing.New(
        ctx,
        "my-service",
        "otel-collector:4317",  // endpoint OTLP/GRPC
        tracing.WithEnvironment("stage"),
        tracing.WithServiceVersion("v1.2.3"),
    )
    if err != nil {
        log.Fatalf("tracer init failed: %v", err)
    }
    defer tracer.Shutdown(ctx)

    ctx, span := tracer.Start(ctx, "MainOp")
    defer span.End()

    span.SetStringAttribute("foo", "bar")
    // ...ваш код...
}
```

---

## Использование

### 1. Создание трейсера

```go
tracer, err := tracing.New(ctx, "service", "localhost:4317", tracing.WithInsecure())
```

Дополнительные опции конструктора:

| Опция                                                | Что делает                                                                                    |
| ---------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| **`WithInsecure()`**                                 | Подключается к OTLP-endpoint без TLS.                                                         |
| **`WithBatchTimeout(d time.Duration)`**              | Максимальное время буферизации батча перед отправкой экспортёру. По-умолчанию — 5 с.          |
| **`WithHostName(name string)`**                      | Записывает `host.name` в ресурс сервиса.                                                      |
| **`WithEnvironment(env string)`**                    | Добавляет атрибут `deployment.environment` (`prod`, `dev`, `stage`, …).                       |
| **`WithServiceVersion(v string)`**                   | Атрибут `service.version` — версия приложения.                                                |
| **`WithSampler(s sdktrace.Sampler)`**                | Задаёт стратегию семплинга (например, `TraceIDRatioBased(0.1)`).                              |
| **`WithResourceAttribute(attr attribute.KeyValue)`** | Добовляет любой произвольный атрибут к ресурсам (`team=backend`, `region=eu-west-1` и т.д.).  |

Все опции комбинируются и передаются в tracing.New(… , opts...).

### 2. Работа со спанами

```go
ctx, span := tracer.Start(ctx, "OperationName")
defer span.End()

span.SetStringAttribute("key", "value")
span.SetIntAttribute("id", 42)
span.SetBoolAttribute("success", true)
span.SetJSONAttribute("payload", map[string]any{"x": 1})

span.AddEvent("Fetched from DB")
span.AddEventWithInt("Items count", "count", 12)

if err != nil {
    span.RecordError(err)
}
```

### 3. gRPC-Interceptors

В клиенте:

```go
conn, _ := grpc.Dial(
    addr,
    grpc.WithStatsHandler(tracing.StatsClientHandler()),
    grpc.WithUnaryInterceptor(tracing.PropagationUnaryInterceptor()),
)
```

В сервере:

```go
server := grpc.NewServer(
    grpc.StatsHandler(tracing.StatsServerHandler()),
    grpc.UnaryInterceptor(tracing.TracingUnaryInterceptor()),
)
```

---

## Интерфейсы

```go
type Tracer interface {
    Start(ctx context.Context, name string) (context.Context, Span)
    TraceIDFromContext(ctx context.Context) string
}

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
```

---

## Интеграция с инфраструктурой

* Совместимо с любыми экспортёрами OpenTelemetry (Jaeger, OTLP, Zipkin и др.)
* Расширяемые опции через паттерн функций-опций
* Не завязывается на конкретную реализацию — легко подменить/замокать в тестах

---

## Зачем этот пакет

* Меньше шаблонного кода вокруг OTel
* Нет утечек зависимостей в core-бизнес-логику
* Легко внедрить tracing в микросервисы с gRPC и HTTP
