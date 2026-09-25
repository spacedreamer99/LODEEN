# Where to resume

## State at freeze (v0.1-frozen)

- Go server: TCP 20Hz, metrics, admin HTTP — works
- Go client: raylib, 6DOF, chat, interpolation — works
- Docker multi-stage scratch (4.5 MB) — works
- docker-compose with PostgreSQL + Redis — works
- CI: GitHub Actions (lint, test, docker, trivy) — green
- GHCR: public image `ghcr.io/spacedreamer99/lodeen-server:main`
- k3d cluster with LODEEN deployed — works

## Next steps (when resuming)

1. Ed25519 identity for players
2. Inventory + resources in world (skeleton of progression)
3. Craft system
4. Building
5. Anti-abuse: PoW, reputation-gate, invites
6. Helm chart + ArgoCD
7. Prometheus + Grafana + SLO
8. Chaos Mesh + Velero
9. Federation between two k3d clusters
10. Launcher

## Concept (full)

See docs/VISION.md (to be written) — but core idea:
- Roguelike civilization, PvE co-op
- Star system, 32 players per world
- Granular linear progression (stages within a world)
- Metaprogression: player's knowledge carries between worlds
- Supernova as wipe + extraction challenge
- KSP-like orbital mechanics
- Bots on client, server syncs + anti-cheat
- Command via physical objects (flags, whistles, campfires, radios)

## Commands to restart

    k3d cluster start lodeen
    kubectl get pods -A
    cd ~/Desktop/LODEEN && make dev

## To resume after diploma

1. Read this file
2. Read README.md
3. Read docs/VISION.md (if exists)
4. Run `make dev` to see it works
5. Start with "Next steps" above
