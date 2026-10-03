# ADR 0002: Recreate strategy для stateful мира

- **Status:** accepted
- **Date:** 2026-10-03

## Context

Мир в памяти. Два pod'а с одним миром = два расходящихся состояния.

## Decision

Стратегия rollout — **Recreate** (или StatefulSet аналог):

1. Старый pod получает SIGTERM
2. Flush snapshot в S3
3. Pod уходит
4. Новый pod стартует
5. Restore из snapshot

Downtime 10–30 сек — приемлемо для PvE.

## Consequences

**+** Никогда два pod'а с одним миром
**+** Snapshot = точка восстановления
**-** Downtime при деплое

## Alternatives

- **RollingUpdate** — невозможен, состояние stateful.
- **Blue-green** — 2 pod'а одновременно → конфликт состояния.
- **Canary** — только с внешним state в БД (не наш случай).
