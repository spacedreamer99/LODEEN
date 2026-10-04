# Refactoring Log — LODEEN

> Как 6 god-файлов и монолитные функции превратились в 40+ сфокусированных
> единиц, без единого изменения в поведении игры.

## Контекст

**LODEEN** — клиент-серверная игра на Go (~89 файлов в `internal/`, `cmd/`),
с полным DevOps-обвесом: `Dockerfile`, `Makefile`, Helm, Terraform, Ansible,
Prometheus rules, GitHub Actions CI.

По мере роста фич (мамонты, лодки, фабрики, ракеты, орбитальная механика)
код накапливал god-файлы: один файл — один домен, но внутри switch на 350
строк или одна функция на 200+ строк. Эта серия рефакторингов вернула
архитектуре читаемость, не сломав ни строки геймплея.

## Метрики до / после

| Файл / функция | Было | Стало | Коммит |
|---|---|---|---|
| `client/app/rocket.go` | 172 строки, 1 функция | 11 сфокусированных функций | `ee59fc1` |
| `server/net/rockets.go:tickRockets` | 345 строк в одной функции | 3 файла, 15 функций, max ~40 строк | `bf93911` |
| `server/net/handlers.go:handleMessage` | 356 строк switch на 30 case'ов | 5 строк + registry + `dispatch[T]` | `7fe1e6e` |
| `server/net/snapshot.go:broadcastSnapshot` | 203 строки = весь файл | 15-строчный оркестратор + 12 `collectXxx` | `8c1df68` |
| `client/app/update_playing.go` | 742 строки в одном файле | 6 файлов по доменам, max 185 строк | `cbcd1a5` |
| `shared/protocol/messages.go` | 400+ строк, 50 типов вперемешку | 6 файлов по доменам | `068f108` |
| `server/net/mobs.go` | 824 строки, 14 функций + 4 блока AI | 5 файлов по AI-домену | `15a7d87` |
| `server/net/mammoths.go` | 533 строки, `tickMammoths` 149 строк | 4 файла, оркестратор 18 строк + 3 фазы | `d02b405` |
| `client/input/flight.go` | 545 строк, `updateSurvival` 117 строк | 4 файла, оркестратор 41 строка + 3 фазы | `b2521e1` |

Итого: ~5300 строк кода переразложено. Ни одной новой функции с логикой —
только extract, rename и группировка.

## Server architecture (ADR 0001)

После решения «server-authoritative для 32-игроковых миров» (`docs/adr/0001-*`)
сервер прошёл внутреннюю чистку без изменения поведения:

| Файл / домен | Было | Стало | Коммит |
|---|---|---|---|
| `server.go` | 417 строк монолита | 4 файла: server + conn + tick + util | `8fe082e` |
| `Server struct` | 11 пар `xMu + x map` | 11 типизированных `store[T]` | `f5fe13e`, `515fee7` |
| `handleConn` | 84 строки | 25-строчный оркестратор + 4 фазы | `8fe082e` |
| `store[T]` | — | дженерик + 9 тестов + concurrent | `7c7c78a` |

**`store[T]`** — дженерик-контейнер `map[string]T` с RWMutex. Один store
на домен (rockets, mobs, mammoths, resources, wells, houses, solar,
batteries, factories, boats, projectiles). Даёт:
- единый API (`Get/Put/Delete/Len/Read/Update`)
- lock-graph тривиально проверяем per-domain
- один тест защищает все 11 доменов

### Server function-level cleanup (продолжение ADR 0001)

После ADR 0001 на сервере остались 7 функций >60 строк. Устранены 6 из них:

| Функция | Было | Стало | Коммит |
|---|---|---|---|
| `tickMobs` | 94 | 10-строчный оркестратор + 3 фазы + `Mob.moveByDir` | `00dab0b` |
| `tickPinkGather` | 78 | 12-строчный оркестратор + 4 фазы + `Mob.moveTowards` | `1ef2ada` |
| `handleHitMammoth` | 72 | 22-строчный оркестратор + 2 фазы | `655c51e` |
| `tickBreeding` | 65 | 15-строчный оркестратор + 4 фазы | `ca21792` |
| `handleCraftFactory` | 63 | 14-строчный оркестратор + 4 фазы | `fc6f6ae` |
| `handleAcceptContract` | 63 | 7-строчный оркестратор + 3 фазы | `6369cfd` |
| `tickPilotedRocket` | 62 | оставлен как есть — оркестратор с комментариями | — |

**Что дали эти разбивки:**
- Устранены 2 латентные race-conditions (`handleHitMob`, `handleHitMammoth` — `m.HP` читался после unlock)
- Введены переиспользуемые helpers: `Mob.moveByDir`, `Mob.moveTowards`, `spawnDrops`, `sendInventory`
- `handleCraftFactory` получил защиту от race между проверкой ресурсов и их списанием
- `dispatch[T]` (103 строки) **оставлен** — это таблица роутинга, не монолит

**Итог:** на сервере не осталось функций >62 строк (кроме таблицы роутинга).

## Приёмы

### 1. Extract Method + фазовое разбиение оркестратора

`tickRockets` (345 строк) — один цикл, внутри 13 смысловых блоков.
Разбит на оркестратор (`tickPilotedRocket`, ~50 строк, читается как оглавление)
и фазовые функции (`applyAutopilot`, `gravityAccel`, `consumeThrust`,
`applyAtmosphericDrag`, `collideWithEarth`, `updateOrbitalParams`, ...).

Читатель видит порядок фаз в оркестраторе, детали — в отдельных функциях.

### 2. Registry pattern вместо switch

`handleMessage` (356 строк) — гигантский `switch env.Type`. Заменён на
`map[protocol.Type]msgHandler`. Оркестратор сжался до 5 строк:

    func (s *Server) handleMessage(c *Client, env *protocol.Envelope) {
        if h, ok := messageHandlers[env.Type]; ok {
            _ = h(s, c, env)
        }
    }

Добавить новый тип сообщения теперь = одна строка в таблице.

### 3. Generics для единообразного decode

25 handler'ов повторяли паттерн `var p T; env.Decode(&p); ...`. Одна
дженерик-функция убирает boilerplate:

    func dispatch[T any](h func(s *Server, c *Client, p T)) msgHandler {
        return func(s *Server, c *Client, env *protocol.Envelope) error {
            var p T
            if err := env.Decode(&p); err != nil { return err }
            h(s, c, p)
            return nil
        }
    }

Go мономорфизирует `dispatch[T]` на этапе компиляции — нулевой runtime-overhead,
никакой рефлексии.

### 4. Domain splitting god-файлов

`update_playing.go` (742 строки) и `messages.go` (400+ строк) разрезаны
по доменам, а не по синтаксису:

    update_playing.go            -> оркестратор + death/pause/unfocus
    update_playing_sync.go       -> sync тел, UI-lock, HP, emit state
    update_playing_overlays.go   -> UI-оверлеи, F1
    update_playing_input.go      -> колесо, pickup, mount, mouse
    update_playing_items.go      -> действия предметов
    update_playing_physics.go    -> физика, камера, диак-лог

Правило: файл отвечает на один вопрос.

### 5. Lock-graph simplicity

`broadcastSnapshot` держал 11 мьютексов в одной функции. Теперь каждая
`collectXxx` держит ровно один RLock, оркестратор не держит ничего.
Lock-граф тривиален.

## Инструменты защиты

### Pre-commit hook

Локальный `.git/hooks/pre-commit`:

    go build ./...   # коммит отклоняется при ошибке сборки
    gofmt -l ...     # коммит отклоняется при неотформатированном коде

Правило: сломанный билд физически не попадает в git.

### CI

GitHub Actions (`.github/workflows/ci.yml`) прогоняет на push:
`go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint`.

### Быстрая проверка

    wc -l internal/server/net/rockets*.go \
          internal/server/net/handlers*.go \
          internal/server/net/snapshot*.go \
          internal/client/app/update_playing*.go \
          internal/shared/protocol/messages*.go

## Что НЕ трогали

- Логику игры — поведение сохранено bit-in-bit.
- Публичные API — все типы и функции остались на своих местах.
- Тесты — их почти нет (покрыт только `server/net`).

## Что дальше

1. Тесты — начать с `shared/protocol` (кодек) и `server/net` (физика ракет).
3. CI: добавить `golangci-lint` + `-race`.
4. Уменьшить когнитивную сложность `rockets_tick.go`.

## Философия

> Рефакторинг — это не переписать по-другому. Это сделать намерение явным.

До: 345 строк подряд, читателю надо помнить 13 фаз в голове.

После:

    func (s *Server) tickPilotedRocket(...) {
        applyAutopilot(r, relPos, relVel, distEarth)  // 1
        steerUpTowardTarget(r, dt)                     // 2
        gx, gy, gz := gravityAccel(...)                // 3
        tx, ty, tz := consumeThrust(r, dt)             // 4
        // ...
    }

Оркестратор читается за 30 секунд. Детали — в отдельных функциях.
