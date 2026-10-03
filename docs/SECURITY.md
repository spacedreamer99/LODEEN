# LODEEN Security — Threat Model

## Что защищаем

- Игровые аккаунты и прогресс
- Сервер (VPS) от компрометации
- Клиент от крашей и RCE
- Платформу (federation) от подмены серверов
- Экономику (inventory, contracts)

## Кого боимся

| Актор | Цель | Вектор |
|---|---|---|
| DDoS | положить сервер | network flood |
| Читер | преимущество в игре | modified client |
| Supply chain | подменить образ/mod | compromised CDN |
| MITM | подменить трафик | public wifi |
| Insider | доступ к чужим данным | RBAC bypass |
| Вредоносный мод | RCE на клиенте | WASM escape |

## Layers

### Network
- TLS 1.3 (cert-manager + Let's Encrypt)
- mTLS между сервисами (позже, если нужно)
- NetworkPolicy: default deny, явные разрешения
- Rate limiting на ingress (nginx)

### Identity
- **Игрок:** Ed25519 keypair в `~/.lodeen/`, подпись challenge при connect
- **Сервер:** Ed25519 fingerprint в `.well-known`
- **CI/деплой:** OIDC (GitHub → GHCR без долгоживущих токенов)

### Secrets
- **k8s:** Sealed-secrets — секреты в git зашифрованные
- **terraform.tfvars:** не в git, `.gitignore`
- **rotation:** Grafana admin каждый 90 дней, SSH-ключи каждый год

### Supply chain
- **gitleaks** в CI — не утекли секреты
- **cosign sign** образы → публикуем только подписанные
- **Admission policy (Kyverno):** отклонить pod без подписи
- **Syft** SBOM → Grype scan
- **pin by digest** — не `:latest`, а `@sha256:...`

### Runtime
- **Falco** — детект аномалий (shell в контейнере, чтение /etc/shadow)
- **audit logs** — кто что делал в k8s API
- **readOnlyRootFilesystem** + drop all caps

### Приложение
- **Input validation** — все сообщения клиента валидируются
- **Anti-cheat** — серверная валидация позиции (макс. дистанция за тик)
- **Rate limit** — max сообщений/сек на игрока
- **Payload size** — MaxFrameSize = 64 KB, отвергаем больше

### Mods (design)
- WASM sandbox (wasmtime)
- Capability-based security
- Manual review перед публикацией
- Revocation list

## Что не в scope (сознательно)

- Защита от гос-уровня
- Защита от quantum-компьютеров (Ed25519 пока ОК)
- DDoS за пределами Cloudflare

## Incident Response

- Report: security@lodeen.example
- SLA: 24 часа на ack, 7 дней на fix
- Disclosure: после fix, через 90 дней

## Roadmap

- [x] Threat model
- [ ] Ed25519 identity игроков
- [ ] Anti-cheat базовая валидация
- [ ] Sealed-secrets + gitleaks (P0)
- [ ] Cosign + Kyverno admission
- [ ] Falco
