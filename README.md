# LODEEN

Кибербезопасность-полигон-игра с собственным движком (клиент + сервер).

## Стек

- Java 17 + LWJGL 3 + OpenGL 3.3 Core
- Gradle (multi-module: client, server, shared)
- Netty - сеть
- Jackson - JSON
- jgltf - загрузка GLB
- ImGui - UI
- Bouncy Castle - крипто
- SQLite + Flyway + HikariCP - БД
- Micrometer + Prometheus - метрики
- Logback + SLF4J - логи

## Сборка

    ./gradlew build              # собрать всё
    ./gradlew :client:run        # запустить клиент
    ./gradlew :server:run        # запустить сервер

## Лицензия

Apache License 2.0 - см. LICENSE.
