# LODEEN Mods — Design

Система модов сообщества. Статус: **design only**, кода нет.

## Модель исполнения: WASM

Почему WASM, а не Go plugins или Lua:
- **Безопасность** — sandbox, capability-based
- **Переносимость** — Linux/Windows/macOS/ARM
- **Детерминизм** — нет races, нет syscalls
- **Инструменты** — wasmtime, wazero (pure Go)

## Mod manifest (mod.json)

    {
      "id": "com.example.starfield",
      "name": "Better Starfield",
      "version": "1.2.0",
      "author": "example",
      "api_version": "1.x",
      "deps": [
        {"id": "lodeen.api.skybox", "range": "^1.0.0"}
      ],
      "capabilities": ["render.read", "config.read"],
      "signature": "<ed25519>",
      "entry": "mod.wasm"
    }

## Capability model

Мод **объявляет** что ему нужно. Пользователь **подтверждает** при установке.

| Capability | Что даёт |
|---|---|
| `world.read` | читать сущности мира |
| `world.write` | создавать/изменять сущности |
| `render.read` | доступ к шейдерам/моделям |
| `net.client` | исходящие HTTP |
| `config.read/write` | своя конфиг секция |

## Загрузка мода

1. Клиент читает mod.json
2. Проверяет подпись
3. Запрашивает подтверждение capabilities у user
4. Загружает WASM в sandbox (wasmtime)
5. Инициализирует с capability-токенами
6. Вызывает `on_enable()`

## Server-side

Сервер решает:
- Какие mods разрешены (whitelist по hash)
- Какие mods обязательны (для геймплея)
- Какие mods только клиентские

При connect клиент отправляет hash-list mods → сервер отвечает allow/deny.

## Mod repository

- CDN с mod-бинарниками
- БД с метаданными (id, version, author, downloads, rating)
- Модерация: ручная проверка модов перед публикацией
- Revocation list: kill-switch для вредоносных модов

## Conflicts

Если 2 мода патчат одну функцию:
- Мод A: hook("on_tick", priority=100)
- Мод B: hook("on_tick", priority=50)
- Вызов по убыванию priority

Если одинаковый priority — пользователь выбирает.

## Что не делаем

- Нейтивные .so/.dll (небезопасно)
- Arbitrary filesystem access
- Моды, меняющие protocol (только client-side)

## Roadmap

- [x] Design
- [ ] WASM runtime integration (wazero)
- [ ] mod.json schema
- [ ] Capability API
- [ ] Mod repository API
- [ ] CLI: lodeen-cli mod install/list/remove
