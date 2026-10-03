# LODEEN — DevOps Portfolio

Федеративная платформа для кооперативного мультиплеера на Go.
Stateful world-сервер, 20 Hz TCP, разворачивается через Helm + ArgoCD,
наблюдается через Prometheus + Grafana.

Ссылки:
- GitHub: https://github.com/spacedreamer99/LODEEN
- Образ: ghcr.io/spacedreamer99/lodeen-server
- Лицензия: Apache 2.0

## Стек

| Слой | Технология |
|---|---|
| Сервер | Go 1.27, TCP, 20 Hz, stateful |
| Клиент | Go + raylib, 6DOF |
| Контейнер | Docker multi-stage, scratch, 4.7 MB |
| Оркестрация | k3d (локально) / k3s (план VPS) |
| Упаковка | Helm chart (deploy/helm/lodeen) |
| GitOps | ArgoCD, Application CR, selfHeal |
| Observability | Prometheus + Grafana + Alertmanager (kube-prometheus-stack) |
| CI | GitHub Actions: lint, test, build, docker, trivy |
| Load testing | cmd/loadtest (headless bots) |

## Архитектура

    ┌──────────────────────────────────┐
    │  GitHub (main)                   │
    │  ├── cmd/                        │
    │  ├── internal/                   │
    │  ├── deploy/helm/lodeen/         │
    │  └── deploy/argocd/application   │
    └────────────┬─────────────────────┘
                 │ git pull
                 ▼
    ┌──────────────────────────────────┐
    │  ArgoCD (k3d)                    │
    │  Application: lodeen             │
    │  syncPolicy: automated + prune   │
    │            + selfHeal            │
    └────────────┬─────────────────────┘
                 │ helm template + kubectl apply
                 ▼
    ┌──────────────────────────────────┐
    │  namespace lodeen                │
    │  Deployment lodeen (1 replica)   │
    │  ├── containerPort 7777 (game)   │
    │  └── containerPort 9091 (admin)  │
    └────────────┬─────────────────────┘
                 │ scrape /metrics
                 ▼
    ┌──────────────────────────────────┐
    │  Prometheus → Grafana            │
    │  ServiceMonitor lodeen           │
    └──────────────────────────────────┘

## Observability

Метрики (Prometheus):

- `lodeen_tick_duration_seconds` — histogram, время тика (SLO: p99 < 5 мс)
- `lodeen_snapshot_bytes` — histogram, размер снапшота
- `lodeen_rtt_seconds` — histogram, RTT клиента (репортит клиент)
- `lodeen_server_players_connected` — gauge
- `lodeen_server_ticks_total` — counter

SLO (проверено нагрузочным тестом):

| Метрика | Цель | Достигнуто (32 бота) |
|---|---|---|
| Tick duration p99 | < 5 мс | 1.0 мс |
| Tick rate | 20 Hz | 20 Hz |
| RTT p99 (loopback) | < 100 мс | ~5 мс |
| Errors | 0 | 0 |

Нагрузочный тест: 32 headless-бота × 60 секунд → 655 snapshots/s, 3.9 MB/s, 0 ошибок.

## GitOps

- Application: deploy/argocd/application.yaml
- Синхронизация автоматическая (prune + selfHeal)
- Реакция на drift: удалённый ресурс восстанавливается за ~15 сек

## Что было сложного

1. Импорт образа в k3d падал на multi-platform attestations → решили через локальный registry (k3d --registry-create).
2. k3d registry не резолвится через CoreDNS → использовали registries.yaml в k3s с http endpoint.
3. CRD applicationsets.argoproj.io не влезал в metadata.annotations → установили через server-side apply.
4. Stateful world нельзя масштабировать в 2+ реплики → Recreate strategy + PDB + описание «один процесс — один мир».


## Alerting

- **PrometheusRule**: 6 алертов
  - LodeenDown, LodeenTickSlow, LodeenRTTHigh, LodeenNoPlayers
  - LodeenSLOFastBurn, LodeenSLOSlowBurn (multi-window multi-burn-rate)
- **Burn rate SLO**: 4 recording rules, Google SRE паттерн
- **Alertmanager**: настроен, готов к Telegram receiver
- **ADR-0004**: почему `absent()` для disappearing series

## Stateful Infrastructure

- **StatefulSet** вместо Deployment: стабильное имя `lodeen-0`
- **PVC** через `volumeClaimTemplates`: `world-data-lodeen-0`
- **Headless Service** для DNS-based discovery
- **updateStrategy: OnDelete** — сами контролируем обновление (stateful)
- **ADR-0001, ADR-0002**: архитектурные решения

## Security

- **CI**: gitleaks (secrets), hadolint (Dockerfile), trivy (vulns)
- **Runtime**: non-root, read-only FS, drop all caps, seccomp
- **Dependabot**: автообновление зависимостей
- **Threat model**: `docs/SECURITY.md`
- **ADR-0003**: local registry для dev vs GHCR для prod

## Operations

- **RUNBOOK.md**: playbooks для типовых инцидентов
- **Postmortem**: `docs/incidents/2026-10-03-argocd-redis-imagepull.md`
- **Blameless culture**: шаблон в `docs/incidents/0000-template.md`

## Design Docs

- **LAUNCHER.md**: авто-обновление клиента, delta updates, signature verify
- **MODS.md**: WASM sandbox, capability-based security
- **FEDERATION.md**: Chain of Trust, .well-known discovery
- **PROTOCOL.md**: semver versioning, fork policy

## ADRs (Architecture Decision Records)

5 решений с обоснованием:
- 0001: StatefulSet вместо Deployment
- 0002: Recreate/OnDelete для stateful мира
- 0003: Local registry для dev vs GHCR
- 0004: absent() в alerting rules
- 0005: Federation — только opt-in

## Что дальше

- **VPS**: Terraform + Ansible bootstrap, k3s на живом сервере
- **cert-manager + ingress**: TLS на VPS
- **Federation**: два k3d кластера, .well-known, Chain of Trust
- **Launcher MVP**: delta updates, signature verify
- **Reputation**: Postgres + API + CLI
- **S3 snapshots + restore**: stateful persistence
- **Telegram alerts**: AlertmanagerConfig
- **Loki + Promtail**: логи рядом с метриками
