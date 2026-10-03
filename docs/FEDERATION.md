# LODEEN Federation — Design

Синхронизация независимых серверов. Статус: **design + skeleton**.

## Концепт

Каждый Server — независимый узел. Синхронизируются через **Chain of Trust** — ручную верификацию.

**Репутация не наследуется.** Новый сервер = явное действие "я ему доверяю".

## Discovery — только opt-in

Никакого active scan. Только:
- `.well-known/lodeen-federation` на домене сервера
- DNS SRV: `_lodeen._tcp.example.com`
- Crawl через уже проверенные сервера

## `.well-known/lodeen-federation`

    {
      "server_id": "abc123...",
      "public_key": "<ed25519>",
      "protocol_versions": ["1.0", "1.1"],
      "worlds": [
        {"id": "survival-eu", "name": "Survival EU", "players": 42, "max": 64}
      ],
      "verified_by": ["server-xyz", "server-abc"]
    }

## Chain of Trust

Ручная верификация:
1. Пользователь вводит URL сервера: `lodeen-cli verify https://example.com`
2. CLI качает `.well-known`, показывает fingerprint
3. Пользователь подтверждает вручную (сравнивает с тем что опубликовано на сайте/форуме)
4. Сервер добавляется в `trusted_servers` в БД

**Никаких автоматических цепочек.** Каждый link — отдельный акт.

## Federation crawl

Периодически (каждые 5 мин):
1. Для каждого trusted server: GET `.well-known/lodeen-federation`
2. Список миров → в каталог (отдельная таблица)
3. Обновление статуса: online/offline, players count

Кэш: 5 мин. Не долбим сервер чаще.

## Каталог миров

Таблица `federated_worlds`:
- `server_id` — откуда пришёл
- `world_id` — id мира на том сервере
- `name`, `players`, `max_players`
- `last_seen` — когда последний раз качали

Когда игрок открывает Launcher — видит все миры со всех trusted серверов.

## Отзыв доверия

`lodeen-cli revoke <server_id>` → сервер удаляется из trusted, его миры пропадают.

Ничего не наследуется. Если сервер А был доверенным, а потом скомпрометировали — отзываем у А, и всё что приходило от А (включая миры сервера Б, которые А ретранслировал) — под вопросом. **Это feature, не bug.**

## Roadmap

- [x] Design
- [ ] `.well-known` HTTP handler
- [ ] CLI: verify/revoke
- [ ] Crawler (5-min poll)
- [ ] Каталог миров
- [ ] Launcher интеграция
