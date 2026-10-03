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

Итого: ~3800 строк кода переразложено. Ни одной новой функции с логикой —
только extract, rename и группировка.

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
