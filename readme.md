# Tracing Package

Простой инструмент для интеграции OpenTelemetry в Go-сервисы с поддержкой gRPC

## Особенности
- Настройка трассировки за 3 шага
- Автоматическая интеграция с gRPC
- Оптимизированная сериализация JSON
- Поддержка распределенного трейсинга
- Гибкая система атрибутов

Установка
```bash
go get github.com/your-repo/tracing
```

## Быстрый старт

### Инициализация:
```go
tw, err := tracing.New(
    context.Background(),
    "my-service",
    "otel-collector:4317",
    tracing.WithEnvironment("prod"),
    tracing.WithServiceVersion("1.2.3"),
)
defer tw.Shutdown(context.Background())
```

### gRPC интеграция:

Сервер:
```go
server := grpc.NewServer(
    grpc.StatsHandler(tracing.StatsServerHandler()),
    grpc.UnaryInterceptor(tracing.TracingUnaryInterceptor()),
)
```
Клиент:
```go
conn, _ := grpc.Dial(
    "localhost:50051",
    grpc.WithStatsHandler(tracing.StatsClientHandler()),
    grpc.WithUnaryInterceptor(tracing.PropagationUnaryInterceptor()),
)
```
### Основные методы

Работа с трейсами:
```go
ctx, span := tw.Start(ctx, "operation")
defer span.End()

tw.SetStringAttribute(span, "user.id", "123")
tw.SetJSONAttribute(span, "request", req)

tw.RecordErrorWithDetails(span, err, map[string]interface{}{
    "attempt":  3,
    "fallback": "cache",
})
```
Извлечение TraceID:
```go
func Handler(w http.ResponseWriter, r *http.Request) {
    traceID := tracing.TraceIDFromContext(r.Context())
    w.Header().Set("X-Trace-ID", traceID)
}
```
### Конфигурация

Доступные опции:
| Метод                     | Описание                               |
|---------------------------|----------------------------------------|
| `WithInsecure()`          | Отключение TLS                         |
| `WithHostName()`          | Имя хоста в метаданных                 |
| `WithEnvironment()`       | Окружение (prod/dev/stage)             |
| `WithServiceVersion()`    | Версия сервиса                         |
| `WithBatchTimeout()`      | Таймаут отправки батчей (default: 5s)  |
| `WithSampler()`           | Кастомная стратегия семплинга          |
| `WithResourceAttribute()` | Произвольные атрибуты ресурсов         |

### Пример интеграции
```go
func ProcessOrder(ctx context.Context, order *Order) error {
    ctx, span := tw.Start(ctx, "ProcessOrder")
    defer span.End()

    tw.SetJSONAttribute(span, "order.details", order)

    if err := validate(order); err != nil {
        tw.RecordError(span, err)
        return err
    }

    return nil
}
```
