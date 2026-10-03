# Contributing to LODEEN

## Quick start

    git clone https://github.com/spacedreamer99/LODEEN.git
    cd LODEEN
    make build
    make dev

## Pre-commit hooks (recommended)

    pip install pre-commit
    pre-commit install

## Code style

- `gofmt` — обязателен
- `golangci-lint run` — обязателен
- Коммиты: `feat(scope): ...`, `fix(scope): ...`, `docs: ...`, `chore: ...`

## Tests

    make test

## Документация

Дизайн-решения — в `docs/adr/`. Каждое большое изменение = новый ADR.
