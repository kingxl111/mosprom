# Graceful Shutdown в ApiGateway

## Обзор

ApiGateway реализует корректное завершение работы (graceful shutdown) для всех компонентов системы. Это обеспечивает:

- Безопасное завершение всех горутин
- Корректное закрытие соединений
- Обработка незавершенных задач
- Логирование статистики при завершении

## Компоненты системы

### 1. HTTP Server
- **Назначение**: Принимает HTTP запросы от клиентов
- **Graceful shutdown**: Ожидает завершения активных соединений (5 сек timeout)
- **Сигналы**: Останавливается при получении SIGINT/SIGTERM

### 2. Event Processor (Kafka Consumer)
- **Назначение**: Обрабатывает события из Kafka с использованием горутин
- **Graceful shutdown**: 
  - Останавливает consumer group
  - Завершает все worker горутины
  - Передает незавершенные задачи другим горутинам
- **Timeout**: 10 секунд для завершения обработки

### 3. Notification Subscriber (Redis Pub/Sub)
- **Назначение**: Подписывается на уведомления через Redis
- **Graceful shutdown**:
  - Закрывает pub/sub соединения
  - Завершает обработку сообщений
- **Timeout**: Немедленное завершение

### 4. Goroutine Monitor
- **Назначение**: Мониторит количество горутин и обнаруживает утечки
- **Graceful shutdown**:
  - Останавливает мониторинг
  - Логирует финальную статистику
- **Timeout**: Немедленное завершение

## Последовательность Graceful Shutdown

### Шаг 1: Получение сигнала остановки
```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
```

### Шаг 2: Остановка Event Processor (10 сек timeout)
```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
if err := eventProcessor.Stop(shutdownCtx); err != nil {
    logger.Error("error during event processor shutdown", slog.Any("err", err))
}
```

**Что происходит:**
- Сигнализируется всем worker'ам о завершении
- Закрывается Kafka consumer group
- Ждем завершения всех горутин через WaitGroup
- При timeout - принудительное завершение

### Шаг 3: Остановка Notification Subscriber
```go
if err := notificationSubscriber.Stop(); err != nil {
    logger.Error("error during notification subscriber shutdown", slog.Any("err", err))
}
```

**Что происходит:**
- Закрываются Redis pub/sub соединения
- Завершается обработка сообщений
- Очищаются callback'и

### Шаг 4: Остановка HTTP Server (5 сек timeout)
```go
httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
    logger.Error("error during server shutdown", slog.Any("err", err))
}
```

**Что происходит:**
- Прекращается прием новых соединений
- Ожидается завершение активных запросов
- При timeout - принудительное закрытие

## Обработка горутин в Event Processor

### Worker горутины
Каждый worker имеет:
- **Уникальный ID**: `worker-0`, `worker-1`, etc.
- **Context handling**: Правильная обработка `context.Done()`
- **Graceful shutdown**: Завершение при получении сигнала
- **Task transfer**: Передача незавершенных задач

```go
func (w *worker) run() {
    for {
        select {
        case <-w.ctx.Done():
            w.processor.logger.Info("worker stopping", slog.String("worker_id", w.id))
            close(w.done)
            return
        case <-w.processor.shutdownChan:
            w.processor.logger.Info("worker shutdown requested", slog.String("worker_id", w.id))
            close(w.done)
            return
        default:
            // Worker ждет сообщения от consumer group
            time.Sleep(100 * time.Millisecond)
        }
    }
}
```

### Consumer Group Handler
- **Setup/Cleanup**: Корректная инициализация и очистка
- **Message processing**: Обработка сообщений с возможностью прерывания
- **Error handling**: Обработка ошибок без падения системы

## Мониторинг горутин

### Отслеживание утечек
- **Порог утечек**: 1000 горутин (настраивается)
- **Проверка каждые 30 секунд**
- **Детальная диагностика** при подозрении на утечку

### Статистика
- Текущее количество горутин
- Максимальное количество
- Общее количество проверок
- Информация о памяти и GC

## Логирование при завершении

```go
logger.Info("server stopped gracefully")
logger.Info("all workers stopped gracefully")
logger.Info("server exited cleanly ✅")
```

## Обработка ошибок

### Timeout при shutdown
- Event Processor: 10 секунд
- HTTP Server: 5 секунд
- Логирование ошибок при принудительном завершении

### Ошибки соединений
- Kafka: Переподключение при ошибках
- Redis: Обработка разрывов соединения
- HTTP: Graceful degradation

## Тестирование Graceful Shutdown

### Ручное тестирование
```bash
# Запуск сервера
go run cmd/server/main.go

# Отправка SIGTERM
kill -TERM <pid>

# Отправка SIGINT (Ctrl+C)
# Автоматически обрабатывается
```

### Docker тестирование
```bash
# Запуск с docker-compose
docker-compose up

# Graceful shutdown
docker-compose down
```

## Заключение

Система graceful shutdown в ApiGateway обеспечивает:

1. **Безопасность**: Корректное завершение всех операций
2. **Надежность**: Обработка ошибок и timeout'ов
3. **Мониторинг**: Отслеживание состояния системы
4. **Логирование**: Полная трассировка процесса завершения

Это критически важно для production среды, где необходимо обеспечить целостность данных и корректную работу всех компонентов системы.
