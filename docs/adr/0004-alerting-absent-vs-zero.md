# ADR 0004: absent() в alerting rules

- **Status:** accepted
- **Date:** 2026-10-03

## Context

Правило `up{job="lodeen"} == 0` **не сработало** когда
deployment схлопнулся до 0 реплик. Series `up{}` не
превращается в 0 — она **исчезает**.

## Decision

    up{job="lodeen"} == 0
    or
    absent(up{job="lodeen"})

Первая часть ловит "target существует но недоступен".
Вторая — "target вообще пропал из скрейпа".

## Consequences

**+** Алерт срабатывает в обоих случаях
**+** Стандартный паттерн в Prometheus-сообществе
**-** Нужно проверять в тестах что не false-positive

## Alternatives

- **`up == 0`** — не работает при отсутствии серии.
- **`count(up{job="lodeen"}) == 0`** — работает, но менее явно.
- **`absent_over_time(up[5m])`** — то же, но требует окно.
