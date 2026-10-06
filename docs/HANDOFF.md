# Handoff — пауза до 2028

## Где остановился
Последний коммит перед freeze: `git log -1`.
Дата: 2026-10-07.

## Как запустить за 5 минут
    cd ~/Desktop/LODEEN
    sudo pkill -9 -f lodeen 2>/dev/null ; sleep 1 ; make dev

Откроются два клиента: pilot1, pilot2. В меню — выбрать ник, адрес (по умолчанию), цвет (16 на выбор), Connect.

## Что сделано
- **Server-authoritative** (ADR 0001): сервер симулирует всё, клиент рисует
- **Тонкий клиент** (ADR 0002): клиент шлёт input, читает snapshot
- **11 доменов на generic `store[T]`** — вместо 11 пар мьютексов
- **9 god-файлов разрезано** (172–824 строк каждый)
- **Все монолиты >62 строк устранены**
- **Player color picker**: 16 цветов в меню → уходит в Hello → сохраняется сервером → видно чужим игрокам
- **Nametags**: ник + цветовой маркер над головой, затухание 25→100м
- **Pre-commit**: build + vet + gofmt + `test -race`
- **CI**: race + coverage + gitleaks + hadolint + trivy + SARIF
- **Coverage**: `shared/protocol` 100%, `shared/version` 100%, `server/net` ~21%

## Что НЕ сделано (roadmap)
- Тесты `server/net` до 40–50% (чистые функции: `tickHostileMob`, `findNearestPlayer`, `collectBreedingCandidates`, `scatteredAround`)
- Интеграционный тест `Server ↔ Client` через `net.Pipe`
- Fuzz на `ReadFrame`
- Процедурный звук (см. заметки про audio-пакет)
- Абстракция ввода для `client/input/flight`

## Что проверить первым делом при возврате
    go test -race ./...
    git --no-pager log --oneline -10
    cat docs/adr/README.md
    cat docs/REFACTOR.md

## Известные проблемы на момент freeze
- Игра запускается, но есть мелкие баги геймплея (по словам автора)
- Звука нет
- Могут быть визуальные артефакты в разных сценах

## Контекст проекта
- **DevOps-портфолио**, не для поиска работы разработчиком
- Стек: Go, raylib, TCP, Prometheus, Docker, Helm, Terraform, Ansible, CI/CD
- Целевой масштаб: 32 игрока в мире, PvE-кооп

## Ссылки
- Repo: github.com/spacedreamer99/LODEEN
- ADR: docs/adr/
- Refactor: docs/REFACTOR.md
- Testing: docs/TESTING.md
