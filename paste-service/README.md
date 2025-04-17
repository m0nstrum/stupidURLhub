# Paste Service

Подсервис для хранения и управления текстовыми блоками (пастами) с возможностью тегирования и генерации уникальных человекочитаемых URL.


## Запуск с использованием Docker

```bash
# Сборка образа
docker build -t paste-service .

# Запуск контейнера
docker run -d -p 8080:8080 \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=paste_service \
  -e TAGGER_BASE_URL=http://tagger-ml:8000 \
  -e SLUGGEN_ADDRESS=slug-generator:50051 \
  --name paste-service paste-service
```

## Конфигурация

Сервис настраивается с помощью переменных окружения:

### Сервер
- `SERVER_PORT` - порт сервера (по умолчанию 8080)
- `SERVER_READ_TIMEOUT` - таймаут чтения запроса (по умолчанию 10s)
- `SERVER_WRITE_TIMEOUT` - таймаут записи ответа (по умолчанию 10s)
- `SERVER_SHUTDOWN_TIMEOUT` - таймаут для graceful shutdown (по умолчанию 5s)
- `SERVER_MAX_REQUEST_SIZE` - максимальный размер запроса (по умолчанию 5MB)
- `SERVER_RATE_LIMIT` - ограничение количества запросов в минуту (по умолчанию 100)
- `TEST_MODE` - запуск в тестовом режиме без базы данных (по умолчанию false)

### База данных
- `DB_HOST` - хост базы данных (по умолчанию localhost)
- `DB_PORT` - порт базы данных (по умолчанию 5432)
- `DB_USER` - пользователь базы данных (по умолчанию postgres)
- `DB_PASSWORD` - пароль базы данных (по умолчанию postgres)
- `DB_NAME` - имя базы данных (по умолчанию paste_service)
- `DB_SSL_MODE` - режим SSL (по умолчанию disable)

### Кэш
- `CACHE_TYPE` - тип кэша: inmemory или redis (по умолчанию inmemory) (я ленивый и пока не сделал редис)
- `REDIS_URL` - URL для подключения к Redis (по умолчанию redis://localhost:6379/0)
- `CACHE_DEFAULT_TTL` - время жизни кэша по умолчанию (по умолчанию 10m)
- `CACHE_GC_INTERVAL` - интервал очистки кэша (по умолчанию 1m)
- `CACHE_REFRESH_TTL` - обновлять TTL при получении из кэша (по умолчанию true)

### Внешние сервисы
- `TAGGER_BASE_URL` - базовый URL сервиса тегирования (по умолчанию http://tagger-ml:8000)
- `TAGGER_TIMEOUT` - таймаут запросов к сервису тегирования (по умолчанию 5s)
- `TAGGER_MAX_TEXT_SIZE` - максимальный размер текста для тегирования (по умолчанию 10KB)
- `SLUGGEN_ADDRESS` - адрес сервиса генерации slug (по умолчанию slug-generator:50051)
- `SLUGGEN_TIMEOUT` - таймаут запросов к сервису генерации slug (по умолчанию 5s)
- `SLUGGEN_MAX_TEXT_SIZE` - максимальный размер текста для генерации slug (по умолчанию 10KB)

## API

### Создание пасты

```
POST /api/pastes

Запрос:
{
  "content": "string",
  "tags": ["string"],
  "expires_in": "1h30m",
  "auto_tag": true
}

Ответ:
{
  "id": "string",
  "slug": "string",
  "content": "string",
  "tags": ["string"],
  "view_count": 0,
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "expires": "timestamp",
  "edit_token": "string"
}
```

### Получение пасты

```
GET /api/pastes/{slug}

Ответ:
{
  "id": "string",
  "slug": "string",
  "content": "string",
  "tags": ["string"],
  "view_count": 0,
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "last_viewed": "timestamp",
  "expires": "timestamp"
}
```

### Обновление пасты

```
PUT /api/pastes/{slug}

Запрос:
{
  "content": "string",
  "tags": ["string"],
  "edit_token": "string"
}

Ответ:
{
  "id": "string",
  "slug": "string",
  "content": "string",
  "tags": ["string"],
  "view_count": 0,
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "last_viewed": "timestamp",
  "expires": "timestamp"
}
```

### Получение популярных паст

```
GET /api/pastes/top?limit=10

Ответ:
[
  {
    "id": "string",
    "slug": "string",
    "content": "string",
    "tags": ["string"],
    "view_count": 0,
    "created_at": "timestamp",
    "updated_at": "timestamp",
    "last_viewed": "timestamp",
    "expires": "timestamp"
  }
]
```

### Получение недавних паст

```
GET /api/pastes/recent?limit=10

Ответ:
[
  {
    "id": "string",
    "slug": "string",
    "content": "string",
    "tags": ["string"],
    "view_count": 0,
    "created_at": "timestamp",
    "updated_at": "timestamp",
    "last_viewed": "timestamp",
    "expires": "timestamp"
  }
]
```

## Тестовый режим

Для запуска сервера в тестовом режиме без подключения к базе данных:

```bash
# Запуск в тестовом режиме
TEST_MODE=true go run main.go
```
(run-test.sh)
В тестовом режиме сервис использует:
- In-memory имплементацию репозитория вместо PostgreSQL
- Мок-клиенты для внешних сервисов

## Лицензия

MIT 