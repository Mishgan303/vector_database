# Реализация ЛР‑2


```mermaid
classDiagram
    class Config {
        +string DataDir
        +uint64 SnapshotEvery
    }
    class VectorDB {
        <<interface>>
    }
    class VectorDBService
    class PointStore {
        <<interface>>
    }
    class MemoryPointStore
    class VectorIndex {
        <<interface>>
    }
    class FlatIndex
    class WAL {
        <<interface>>
    }
    class TidwallWALAdapter
    class SnapshotStore {
        <<interface>>
    }
    class FileSnapshotStore
    VectorDB <|.. VectorDBService
    PointStore <|.. MemoryPointStore
    VectorIndex <|.. FlatIndex
    WAL <|.. TidwallWALAdapter
    SnapshotStore <|.. FileSnapshotStore
    VectorDBService --> PointStore
    VectorDBService --> VectorIndex
    VectorDBService --> WAL : общий для БД
    VectorDBService --> SnapshotStore : общий для БД
```

`Open` создаёт адаптеры и сервис, загружает snapshot и WAL, строит индексы и возвращает `VectorDB`.
`Config.DataDir` обязателен; `SnapshotEvery = 0` выбирает 1000 операций.
Один каталог использует один экземпляр БД.

Методы коллекций, `UpsertBatch`, `Get`, `DeleteBatch`, `Search` и `Close` реализованы в отдельных файлах `internal/core/service`. Публичные `core.Open`, `core.Config` и `core.VectorDB` остаются в `internal/core`.
Число коллекций может быть больше одного, но WAL, его последовательность LSN и snapshot общие для локальной БД.
