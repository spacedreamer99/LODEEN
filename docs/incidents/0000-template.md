# Incident YYYY-MM-DD: <краткое название>

- **Severity:** SEV1 | SEV2 | SEV3
- **Duration:** HH:MM – HH:MM (XX мин)
- **Impact:** кто и что пострадало
- **Author:** <имя>
- **Status:** draft | reviewed | closed

## Summary

Один абзац: что случилось, каков impact, сколько длилось.

## Timeline (MSK)

| Time | Event |
|---|---|
| HH:MM | Alert fired |
| HH:MM | On-call ack |
| HH:MM | Root cause found |
| HH:MM | Fix deployed |
| HH:MM | Service restored |
| HH:MM | Alert resolved |

## Root Cause

Что реально произошло. Технические детали. Без обвинений.

## Detection

Как узнали? Alert? Пользователь? Метрика?

## Resolution

Что сделали чтобы восстановить.

## Action Items

- [ ] Что сделать чтобы не повторилось — owner — deadline

## Lessons Learned

Что узнали. Что было хорошо. Что можно улучшить.

## Links

- Grafana: ...
- Prometheus alert: ...
- ArgoCD: ...
- Git commits: ...
