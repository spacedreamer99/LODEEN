# Architecture Decision Records

Формат: [MADR](https://adr.github.io/madr/).
Каждое решение — отдельный файл `NNNN-title.md`.

| # | Решение | Статус | Дата |
|---|---|---|---|
| [0001](0001-server-authoritative.md) | Server-authoritative для миров на 32 игрока | Accepted | 2026-10-04 |
| [0002](0002-client-server-contract.md) | Контракт клиент↔сервер (тонкий клиент) | Accepted | 2026-10-04 |

## Краткая сводка

### ADR 0001 — Server-authoritative
**Что решено:** сервер симулирует всё, клиент рисует.
**Почему:** 32-игроковые PvE-миры не требуют client-side prediction; anti-cheat by construction,
нет desync, reconnect из коробки.
**Что НЕ выбираем:** thin-client, host-client, P2P.

### ADR 0002 — Client-server contract
**Что решено:** клиент шлёт только `*Input` команды + читает snapshot. Не симулирует.
**Почему:** одна истина на всех, никаких рассинхронов.
**Проверка соответствия:** клиент не изменяет поля снапшота (Fuel/Apoapsis и т.п. — read-only).
