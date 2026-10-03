# ADR 0003: Local registry для dev, GHCR для prod

- **Status:** accepted
- **Date:** 2026-10-03

## Context

k3d image import падает на multi-platform attestations
(ошибка `content digest not found`). Нужен надёжный способ
доставить образы в кластер.

## Decision

- **Dev (k3d):** локальный registry через `k3d --registry-create`
- **Prod (VPS/k3s):** GHCR (публичный образ)

k3d cluster.create: `--registry-create lodeen-registry:5000`

## Consequences

**+** Импорт работает без attestations
**+** Dev повторяет prod-паттерн (pull из registry)
**-** Локальный registry — dev-only, при пересоздании кластера теряется

## Alternatives

- **k3d image import** — падает на attestations.
- **kind** — те же проблемы с containerd.
- **Только GHCR** — невозможно без сети, медленно.
