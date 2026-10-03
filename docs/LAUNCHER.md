# LODEEN Launcher — Design

Авто-обновление клиента, разрешение зависимостей, каталог миров.
Статус: **design + MVP skeleton** (cmd/launcher).

## Компоненты

1. **Manifest** — JSON на CDN: версия, список файлов, хеши, подпись
2. **Downloader** — параллельная загрузка, resume, rate-limit
3. **Delta** — zstd --patch-from (скачиваем только diff)
4. **Verifier** — Ed25519 подпись manifest + SHA256 файлов
5. **Installer** — atomic replace, symlink current, rollback
6. **Resolver** — MVS (minimal version selection) для deps
7. **Catalog** — список миров (fetched из federation)

## Manifest

    {
      "version": "1.4.2",
      "channel": "stable",
      "min_launcher": "1.0.0",
      "files": [
        {"path": "bin/lodeen-client", "sha256": "...", "size": 12345678},
        {"path": "assets/planet.glb", "sha256": "...", "size": 456789}
      ],
      "signature": "<ed25519>",
      "pubkey_id": "lodeen-official-2026"
    }

## Установка

    versions/
      1.4.1/   ← старая
      1.4.2/   ← новая
    current -> versions/1.4.2

Проверка перед switch:
- все файлы скачаны и SHA256 совпал
- подпись manifest валидна
- бинарь запускается с --version успешно

## Rollback

Если новая версия крашится 3 раза подряд:
- ~/.lodeen/crash-count
- launcher автоматически переключает current на versions/1.4.1
- отчёт в telemetry (opt-in)

## Каналы

- **stable** — релизы раз в 2 недели
- **beta** — фичи готовые, но не проверенные на 100%
- **nightly** — из main ветки, для разработчиков

## Dependencies

MVS как в Go modules:
- каждый mod/asset объявляет min_version
- resolver выбирает максимальную из минимальных
- lockfile фиксирует точные версии

## Что не делаем (сознательно)

- Свой CDN — Cloudflare/Yandex
- Свой OAuth — не нужно
- P2P — не нужно
- GUI launcher — CLI + JSON достаточно

## Roadmap

- [x] Design
- [ ] Manifest struct + signature verify
- [ ] Downloader с resume
- [ ] Delta updates
- [ ] Atomic install + rollback
- [ ] Resolver (MVS)
- [ ] Catalog + federation
