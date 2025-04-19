# Paste Service

Подсервис для хранения и управления текстовыми пастами с возможностью тегирования и генерации уникальных человекочитаемых URL.


## Запуск с использованием Docker

```bash
# Сборка образа
docker build -t paste-service .

# Запуск контейнера
docker run -d -p 8080:8080 \
  -e DATABASE_HOST=localhost \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=postgres \
  -e DATABASE_PASSWORD=postgres \
  -e DATABASE_DBNAME=paste_service \
  -e DATABASE_SSLMODE=disable \
  -e CACHE_TYPE=redis \
  -e CACHE_REDISURL=redis://localhost:6379/0 \
  -e TAGGER_BASEURL=http://tagger-ml:8000 \
  -e SLUGGEN_ADDRESS=slug-generator:50051 \
  --name paste-service paste-service
```

## Запуск с использованием Docker Compose

Для запуска сервиса вместе с PostgreSQL и Redis, используйте Docker Compose:

```bash
# Запуск всех сервисов
docker-compose up -d

# Остановка сервисов
docker-compose down

# Остановка сервисов и удаление данных
docker-compose down -v
```

Сервис будет доступен на порту 8080, PostgreSQL на 5432, Redis на 6379.

## Конфигурация

Сервис настраивается с помощью переменных окружения:

### Сервер
- `SERVER_PORT` - порт сервера (по умолчанию 8080)
- `SERVER_READTIMEOUT` - таймаут чтения запроса (по умолчанию 10s)
- `SERVER_WRITETIMEOUT` - таймаут записи ответа (по умолчанию 10s)
- `SERVER_SHUTDOWNTIMEOUT` - таймаут для graceful shutdown (по умолчанию 5s)
- `SERVER_MAXREQUESTSIZE` - максимальный размер запроса (по умолчанию 5MB)
- `SERVER_RATELIMIT` - ограничение количества запросов в минуту (по умолчанию 100)
- `SERVER_TESTMODE` - запуск в тестовом режиме без базы данных (по умолчанию false)

### База данных
- `DATABASE_HOST` - хост базы данных (по умолчанию localhost)
- `DATABASE_PORT` - порт базы данных (по умолчанию 5432)
- `DATABASE_USER` - пользователь базы данных (по умолчанию postgres)
- `DATABASE_PASSWORD` - пароль базы данных (по умолчанию postgres)
- `DATABASE_DBNAME` - имя базы данных (по умолчанию paste_service)
- `DATABASE_SSLMODE` - режим SSL (по умолчанию disable)

### Кэш
- `CACHE_TYPE` - тип кэша: inmemory или redis (по умолчанию inmemory)
- `CACHE_REDISURL` - URL для подключения к Redis (по умолчанию redis://localhost:6379/0)
- `CACHE_DEFAULTTTL` - время жизни кэша по умолчанию (по умолчанию 10m)
- `CACHE_GCINTERVAL` - интервал очистки кэша (по умолчанию 1m)
- `CACHE_REFRESHTTLONGET` - обновлять TTL записи при получении ее из кэша (по умолчанию true, пытаемся держать популярные записи в кэше и не перезакидывать их лишний раз)

### Внешние сервисы
- `TAGGER_BASEURL` - базовый URL сервиса тегирования (по умолчанию http://tagger-ml:8000)
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