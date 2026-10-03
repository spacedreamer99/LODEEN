# ADR 0005: Federation — только opt-in

- **Status:** accepted
- **Date:** 2026-10-03

## Context

Как серверы находят друг друга? Варианты:
- Active scan (обход интернета)
- Central registry
- Opt-in через `.well-known`

## Decision

**Только opt-in.** Никакого active scan.

- `.well-known/lodeen-federation` на домене сервера
- DNS SRV: `_lodeen._tcp.example.com`
- Crawl через уже проверенные сервера

Ручная верификация каждого link (Chain of Trust).

## Consequences

**+** Безопасно — никакого шума в сети
**+** Репутация не наследуется автоматически
**+** Сервер сам решает кого пускать
**-** Медленный рост сети (opt-in требует явного действия)
**-** Сложнее bootstrapping первых серверов

## Alternatives

- **Central registry** — single point of failure, цензура.
- **Active scan** — неэтично, шум, fingerprinting.
- **DHT (BitTorrent-style)** — overkill, сложно.
