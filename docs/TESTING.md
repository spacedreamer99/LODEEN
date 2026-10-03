# Testing

## Быстрый прогон

    make test              # go test ./...
    go test -race ./...    # с race-detector (обязательно перед push)
    go test -cover ./...   # краткая сводка по покрытию

Pre-commit hook автоматически прогоняет build + vet + gofmt + test -race
на каждый коммит. Сломанный билд физически не попадает в git.

CI (.github/workflows/ci.yml) на каждый push:
- gofmt (запрещает неотформатированный код)
- go vet ./...
- golangci-lint (v2.13.0)
- go test -race -coverprofile=coverage.out ./...
- gitleaks, hadolint, trivy (SARIF upload)
- docker build + push в GHCR

## Покрытие

| Пакет | Coverage | Что покрыто |
|---|---|---|
| shared/protocol | 100.0% | кодек Envelope, фрейминг, геометрия террейна |
| shared/version | 100.0% | Get/String контракты |
| server/net | 14.6% | орбитальная физика, парсер команд |
| client/*, cmd/* | 0% | рендер/окно/raylib — вне scope юнит-тестов |

Проверить локально:

    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1
    go tool cover -func=coverage.out | grep -v '100.0%'

## Что именно покрыто

### shared/protocol/frame.go — фрейминг
- Round-trip всех размеров (0, 1, 1 KB, ровно MaxFrameSize)
- Отказ от OOM: length = 4 GB отвергается до make([]byte, n)
- Границы MaxFrameSize (64 KB): ровно — ок, +1 — ErrFrameTooLarge
- Битый header (< 4 байт) -> io.ErrUnexpectedEOF
- Битый body (length больше данных) -> io.ErrUnexpectedEOF
- Пустой reader -> io.EOF
- 3 фрейма в одном stream читаются подряд без потери байт
- Ошибка writer'а пробрасывается наружу

### shared/protocol/terrain.go — рельеф
- TerrainHeight детерминирована, ограничена [PlanetRadius +- MaxRelief]
- SurfaceRadius никогда не ниже SeaLevel, scale-invariant
- ClampToSurface сохраняет направление, поднимает точки из-под поверхности

### shared/protocol/messages.go — Envelope
- Round-trip всех публичных типов (PlayerState, Rocket, Snapshot, Chat, ...)
- Decode пустого payload не падает
- Битый JSON возвращает ошибку
- NewEnvelope с math.NaN() возвращает ошибку marshal

### server/net/rockets_orbit.go — орбитальная механика
- Круговая орбита: apo == peri == alt, speed == targetV == v_circ
- Эллипс с известными a, e: apo/peri совпадают с vis-viva
- Параболическая и гиперболическая орбиты -> (0, 0)
- r < 1 (рядом с центром) -> (0, 0) без паники
- Translation invariance: сдвиг primaryPos не меняет результат

### server/net/handlers_chat.go — парсер /get
- Table-driven: 10 кейсов (stone10, a0, x999999999999, пусто, ...)
- Граница clamp: x100000 не клампится, x100001 клампится до 100000

## Инварианты, найденные тестами

### ClampToSurface не строго идемпотентна
Первый вызов: (30.022213, ...). Второй: (30.022217, ...).
Дрейф ~1e-6 из-за float32-нормализации. Инвариант ослаблен до |delta| < 1e-5
и задокументирован в тесте — вместо требования невозможного бит-равенства.

### ReadFrame защищён от OOM-атак
Наивная реализация make([]byte, n) при n = 0xFFFFFFFF укладывает процесс.
Текущая проверяет n > MaxFrameSize ДО аллокации. Тест
TestReadFrame_OverLimit фиксирует это поведение.

## Что НЕ покрыто и почему

- server/net/tick* — требует запущенного *Server с мьютексами и map'ами.
  Покрывать надо интеграционными тестами, а не юнит-тестами.
- client/render/* — рендер raylib в headless CI не проверишь.
- client/input/* — физика завязана на raylib global state (rl.IsKeyDown),
  для юнит-тестов нужна абстракция ввода. Отдельная задача.
- cmd/* — тонкие обёртки над main, 0% приемлем.

## Roadmap

- [ ] Интеграционные тесты на Server + Client через net.Pipe
- [ ] Абстракция ввода для тестов client/input/flight (survival physics)
- [ ] Fuzz-тесты на ReadFrame (go test -fuzz)
- [ ] Codecov / Go Report Card badge в README
