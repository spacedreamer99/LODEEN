# LODEEN Protocol — Versioning & Fork Policy

## Формат сообщения

- Frame: 4 байта length (big-endian) + JSON payload
- Max frame: 64 KB
- Все типы сообщений — в `internal/shared/protocol/`

## Версионирование

`MAJOR.MINOR.PATCH`:

- **MAJOR** — breaking change. Старый клиент **не подключается**.
- **MINOR** — new features, обратно совместимо.
- **PATCH** — bugfixes, совместимо.

## Handshake

    Client → Server: Hello { nick, version, features[] }
    Server → Client: Welcome { player_id, server_version, features[] }

Обе стороны объявляют **features**:
- `chat.v1`, `rocket.v1`, `inventory.v1`, `federation.v1`
- Server может отклонить клиент если нет **обязательной** feature

## Graceful degradation

Если клиент не поддерживает feature — сервер работает без неё.
Пример: клиент без `federation.v1` не получает список чужих миров.

## Fork Policy

**Форк обязан сохранить:**
- Формат Frame (4-byte length + JSON)
- Handshake protocol
- Базовые типы: Hello, Welcome, Snapshot, State, Ping, Pong
- Semver правила

**Форк может менять:**
- Геймплей, физику, механику
- Дополнительные сообщения (с префиксом `<fork>.`)
- Формат Snapshot contents (если MAJOR bump)

**Форк обязан опубликовать:**
- `/.well-known/lodeen-fork` с описанием protocol версии и capabilities
- Свой публичный ключ подписи

## Совместимость

Клиент official ↔ Сервер fork:
- Если MAJOR совпадает — работает
- Если MINOR отстаёт — работает с warnings
- Если MAJOR не совпадает — отказ с понятным сообщением

## Roadmap

- [x] Design
- [ ] Semver enforcement в handshake
- [ ] Feature negotiation
- [ ] `.well-known/lodeen-fork` handler
- [ ] Compatibility matrix
- [ ] Тесты совместимости old client ↔ new server
