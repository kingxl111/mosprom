# ApiGateway

API Gateway для системы сбора, консолидации, анализа и визуализации данных предприятий Москвы.

## Архитектура

ApiGateway построен по аналогии с AuthService и включает следующие компоненты:

### Основные компоненты

1. **HTTP Server** - принимает запросы от клиентов
2. **AuthService Proxy** - проксирует запросы авторизации к AuthService
3. **Event Processor** - обрабатывает события из Kafka с использованием горутин
4. **Notification Subscriber** - подписывается на уведомления через Redis pub/sub
5. **Goroutine Monitor** - мониторит количество горутин и обнаруживает утечки

### Система обработки событий

#### Kafka Consumer с горутинами

- **Producer горутина**: читает события из Kafka
- **N Consumer горутин**: настраиваемое количество worker'ов для обработки
- **Уникальные ID**: каждая горутина имеет уникальный ID для трассировки
- **Context handling**: правильная обработка `context.Done()`
- **Graceful shutdown**: завершение при получении сигналов остановки
- **Task transfer**: передача незавершенных задач другим горутинам

#### Redis Pub/Sub

- **Каналы уведомлений:
  - `file_uploaded` - файл загружен
  - `file_processing` - файл в обработке
  - `file_processed` - файл обработан
  - `file_failed` - ошибка обработки

### Graceful Shutdown

Система поддерживает корректное завершение работы:

1. **Обработка сигналов**: SIGINT, SIGTERM
2. **Последовательность остановки**:
   - Event Processor (10 сек timeout)
   - Notification Subscriber
   - HTTP Server (5 сек timeout)
3. **WaitGroup**: ожидание завершения всех горутин
4. **Логирование статистики** при завершении

### Мониторинг горутин

- **Активное количество**: отслеживание текущего количества горутин
- **Обнаружение утечек**: порог для выявления потенциальных утечек
- **Статистика**: максимум, общее количество проверок
- **Детальная диагностика**: при подозрении на утечку

## Конфигурация

Скопируйте `env.example` в `.env` и настройте переменные:

```bash
cp env.example .env
```

### Переменные окружения

- `HTTP_HOST`, `HTTP_PORT` - настройки HTTP сервера
- `LOG_LEVEL` - уровень логирования
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` - настройки Redis
- `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_GROUP_ID`, `KAFKA_WORKERS` - настройки Kafka
- `AUTH_SERVICE_HOST`, `AUTH_SERVICE_PORT` - настройки AuthService
- `JWT_SECRET` - секретный ключ для JWT

## Запуск

```bash
go run cmd/server/main.go
```

## API Endpoints

### Авторизация (прокси к AuthService)
- `POST /api/v1/auth/register` - регистрация
- `POST /api/v1/auth/login` - вход
- `POST /api/v1/auth/refresh` - обновление токена
- `POST /api/v1/auth/logout` - выход
- `GET /api/v1/auth/me` - информация о пользователе

### Обработка файлов
- `POST /api/v1/files/upload` - загрузка файла
- `GET /api/v1/files/status/{file_id}` - статус обработки файла

## Поток обработки файлов

1. **Клиент загружает файл** → `POST /api/v1/files/upload`
2. **ApiGateway сохраняет информацию** в Redis
3. **Файл отправляется** в Kafka для обработки
4. **Event Processor** обрабатывает файл в фоне
5. **Уведомление о готовности** публикуется в Redis
6. **Клиент может проверить статус** → `GET /api/v1/files/status/{file_id}`

## Структура проекта

```
ApiGateway/
├── cmd/server/main.go          # Точка входа
├── internal/
│   ├── config/                 # Конфигурация
│   ├── environment/            # HTTP окружение
│   ├── event/                  # Обработка событий Kafka
│   ├── gates/http-server/     # HTTP обработчики
│   ├── monitoring/             # Мониторинг горутин
│   ├── notification/           # Redis pub/sub
│   ├── proxy/                  # Прокси к AuthService
│   └── redis/                  # Redis клиент
├── go.mod
├── go.sum
└── env.example
```

## Зависимости

- **Kafka**: IBM/sarama для работы с Kafka
- **Redis**: go-redis/redis для pub/sub
- **HTTP**: стандартная библиотека Go
- **Logging**: log/slog
- **Graceful shutdown**: golang.org/x/sync/errgroup
