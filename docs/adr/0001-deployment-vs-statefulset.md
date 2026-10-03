# ADR 0001: StatefulSet вместо Deployment для мира

- **Status:** accepted (2026-10-03)
- **Deciders:** project owner

## Context

Мир — stateful процесс. Один мир = один pod. Нужно
стабильное имя, persist PVC, порядок старта/остановки.

## Decision

Используем **StatefulSet** с `volumeClaimTemplates`.

- Pod: `lodeen-0`
- PVC: `data-lodeen-0`
- podManagementPolicy: OrderedReady

## Consequences

**+** Стабильное имя → DNS, Service targeting
**+** PVC переезжает с подом
**+** Порядок старта для будущих реплик (миров)
**-** Recreate strategy заменяется на StatefulSet-логику

## Alternatives

- **Deployment + PVC** — работает, но имя пода меняется; PVC ручной.
- **DaemonSet** — только один под на ноду, не наш случай.
