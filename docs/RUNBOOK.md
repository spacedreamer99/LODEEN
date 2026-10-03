# LODEEN Runbook

Что делать при типовых инцидентах.

## Порты dev

- 3000 — Grafana
- 9090 — Prometheus
- 9093 — Alertmanager
- 8081 — ArgoCD UI
- 7777 — LODEEN game
- 19091 — LODEEN admin/metrics

Поднять всё: `bash scripts/pf.sh`

---

## Инцидент: LODEEN недоступен

**Симптом:** `curl localhost:19091/healthz` не отвечает. Алерт `LodeenDown | firing`.

### Диагностика

    kubectl -n lodeen get statefulset
    kubectl -n lodeen get pods
    kubectl -n lodeen describe pod lodeen-0
    kubectl -n lodeen logs lodeen-0 --tail=50
    kubectl -n argocd get application lodeen

### Типовые причины

1. **Pod CrashLoopBackOff** — ошибка в новой версии. Откат через ArgoCD History или `git revert`.
2. **ImagePullBackOff** — образ не доступен. Проверь `docker exec k3d-lodeen-server-0 crictl images`.
3. **PVC не Bound** — проблема со storage class. `kubectl -n lodeen get pvc`.
4. **ArgoCD OutOfSync** — смотреть `status.operationState.message`, форс через UI с PRUNE.

---

## Инцидент: Tick p99 > 5 мс

**Симптом:** `LodeenTickSlow | firing`, burn rate alerts.

### Диагностика

    curl -s 'http://localhost:9090/api/v1/query?query=histogram_quantile(0.99,sum%20by(le)(rate(lodeen_tick_duration_seconds_bucket[5m])))' | python3 -m json.tool
    kubectl -n lodeen top pod lodeen-0
    curl -s 'http://localhost:9090/api/v1/query?query=lodeen_server_players_connected'

### Причины

1. Много игроков + мобов — тяжёлая физика
2. Утечка горутин — `go_goroutines` растёт линейно
3. GC паузы — pprof

---

## Инцидент: ArgoCD застрял в OutOfSync

1. Смотреть ошибку:

    kubectl -n argocd get application lodeen -o jsonpath='{.status.operationState.message}'

2. Если ошибка в манифесте (типа `Recreate` в StatefulSet) — фикс в git, push, refresh:

    kubectl -n argocd annotate application lodeen argocd.argoproj.io/refresh=hard --overwrite

3. Если retry исчерпан — сбросить operationState:

    kubectl -n argocd patch application lodeen --type=json \
      -p='[{"op":"remove","path":"/status/operationState"}]'

4. Форсировать через UI: `https://localhost:8081` → lodeen → SYNC → ✅ PRUNE

---

## Инцидент: k3d кластер не работает

    k3d cluster list
    k3d cluster start lodeen
    kubectl get nodes
    kubectl get pods -A

Если node NotReady:

    docker ps | grep k3d
    k3d cluster stop lodeen && k3d cluster start lodeen

---

## Полезные команды

    # Все алерты
    curl -s 'http://localhost:9090/api/v1/alerts' | python3 -m json.tool

    # Правила
    curl -s 'http://localhost:9090/api/v1/rules' | python3 -m json.tool

    # Targets
    curl -s 'http://localhost:9090/api/v1/targets' | python3 -m json.tool

    # ArgoCD app
    kubectl -n argocd get application lodeen

    # Helm
    helm -n lodeen list
    helm -n lodeen history lodeen

## После инцидента

Postmortem в `docs/incidents/YYYY-MM-DD-<name>.md`. Шаблон — `docs/incidents/0000-template.md`.
