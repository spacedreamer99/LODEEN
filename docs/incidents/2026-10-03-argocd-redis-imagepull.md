# Incident 2026-10-03: ArgoCD Redis ImagePullBackOff

- **Severity:** SEV3
- **Duration:** ~40 минут
- **Impact:** ArgoCD pod `argocd-redis` не запускался. UI работал в degraded mode.
- **Author:** project owner
- **Status:** closed

## Summary

После пересоздания k3d кластера образ `redis:8.2.3-alpine` не смог загрузиться.
Причина — `k3d image import` падал на multi-platform attestations. Решили через локальный registry.

## Timeline (MSK)

| Time | Event |
|---|---|
| 10:55 | Пересоздали k3d кластер |
| 11:00 | Redis pod в ImagePullBackOff |
| 11:05 | `k3d image import` — ошибка `content digest not found` |
| 11:15 | Создали k3d registry через `--registry-create` |
| 11:20 | Push redis в локальный registry |
| 11:25 | Прописали mirror в `registries.yaml` |
| 11:30 | Рестарт k3s — всё ещё ErrImagePull |
| 11:35 | Переключили deployment на `lodeen-registry:5000/redis` |
| 11:40 | Redis Running, ArgoCD работает |

## Root Cause

Три причины в каскаде:

1. **k3d image import bug** — не поддерживает OCI attestation manifests (SBOM, provenance), которые генерирует Docker BuildKit.

2. **CoreDNS не знает docker-имена** — `lodeen-registry` резолвится только из docker network, не из pod network.

3. **containerd идёт по HTTPS** — без `registries.yaml` с endpoint `http://...` падает.

## Detection

Pod в ImagePullBackOff — видно в `kubectl get pods`.

## Resolution

    # 1. Создать registry (при создании кластера)
    k3d cluster create lodeen --registry-create lodeen-registry:5000

    # 2. Push образ
    docker tag redis:8.2.3-alpine localhost:5000/redis:8.2.3-alpine
    docker push localhost:5000/redis:8.2.3-alpine

    # 3. Прописать mirror в k3s
    docker exec k3d-lodeen-server-0 sh -c 'cat > /etc/rancher/k3s/registries.yaml <<EOM
    mirrors:
      lodeen-registry:5000:
        endpoint:
        - http://lodeen-registry:5000
    EOM'
    docker exec k3d-lodeen-server-0 systemctl restart k3s

    # 4. Патч deployment
    kubectl -n argocd set image deployment/argocd-redis \
      redis=lodeen-registry:5000/redis:8.2.3-alpine

## Action Items

- [x] Создать локальный k3d registry
- [x] Прописать mirror в registries.yaml
- [x] ADR-0003 (local registry vs GHCR)
- [ ] В prod использовать GHCR, не k3d image import
- [ ] Скрипт `scripts/k3d-create.sh` для полного provisioning

## Lessons Learned

**Хорошо:**
- Быстро нашли workaround
- ArgoCD UI работал без redis (degraded mode)

**Плохо:**
- 40 минут на исследование k3d bug
- Нет автоматизации registry при создании кластера

## Links

- ADR-0003: `docs/adr/0003-local-registry-vs-ghcr.md`
