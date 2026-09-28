# Интерфейсы векторной БД

## Компоненты

```mermaid
flowchart TD
    Client[Клиент] --> DB[VectorDB]
    DB --> Store[PointStore]
    DB --> Index[VectorIndex]
    DB --> WAL[WAL]
    DB --> Snapshot[SnapshotStore]
    DB --> Router[ShardRouter]
    WAL --> Replica[Replicator]
```

`VectorDB` координирует сценарии, а остальные интерфейсы изолируют способы хранения, поиска, восстановления и распределения данных.

## Основные типы и интерфейсы

```mermaid
classDiagram
    class VectorDB {
        <<interface>>
        +CreateCollection(ctx, config) error
        +DeleteCollection(ctx, name) error
        +GetCollection(ctx, name) CollectionConfig, error
        +ListCollections(ctx) CollectionConfig[], error
        +UpsertBatch(ctx, collection, points) BatchResult, error
        +Get(ctx, collection, ref) VectorPoint, error
        +DeleteBatch(ctx, collection, refs) BatchResult, error
        +Search(ctx, collection, query) SearchResult[], error
        +Close(ctx) error
    }

    class CollectionConfig {
        +string Name
        +int Dimension
        +DistanceMetric Metric
        +int ShardCount
        +int ReplicationFactor
    }

    class VectorPoint {
        +string ID
        +string PartitionKey
        +string* ParentID
        +float32[] Vector
        +string Content
        +map Metadata
    }

    class PointUpsert {
        +string ID
        +string PartitionKey
        +string* ParentID
        +float32[]* Vector
        +string* Content
        +map MetadataPatch
        +string[] DeleteMetadataKeys
    }

    class PointRef {
        +string ID
        +string PartitionKey
    }

    class SearchQuery {
        +float32[] Vector
        +int TopK
        +SearchMode Mode
        +Filter* Filter
        +bool IncludeVector
    }

    class SearchResult {
        +PointRef Ref
        +string* ParentID
        +string Content
        +map Metadata
        +float32 Score
        +float32[] Vector
    }

    class Filter {
        +Condition[] Conditions
    }

    class Condition {
        +string Field
        +FilterOperator Operator
        +any Value
    }

    class BatchResult {
        +PointRef[] Applied
        +BatchFailure[] Failed
    }

    VectorDB ..> CollectionConfig
    VectorDB ..> PointUpsert
    VectorDB ..> PointRef
    VectorDB ..> VectorPoint
    VectorDB ..> SearchQuery
    VectorDB ..> SearchResult
    VectorDB ..> BatchResult
    VectorPoint --> PointRef : identity
    SearchQuery --> Filter
    Filter *-- Condition
```

## Вспомогательные структуры

```mermaid
classDiagram
    class BatchFailure {
        +PointRef Ref
        +DBError Error
    }
    class DBError {
        +ErrorCode Code
        +string Message
        +map Details
        +Error() string
    }
    class Neighbor {
        +PointRef Ref
        +float32 Score
    }
    class IndexSearchRequest {
        +float32[] Vector
        +int TopK
        +SearchMode Mode
        +set AllowedRefs
    }
    class WALOperation {
        +WALOperationType Type
        +string Collection
        +byte[] Payload
    }
    class WALEntry {
        +LSN LSN
        +WALOperation Operation
    }
    class Snapshot {
        +LSN LastLSN
        +byte[] Payload
    }
    BatchResult *-- BatchFailure
    BatchFailure --> DBError
    BatchFailure --> PointRef
    Neighbor --> PointRef
    IndexSearchRequest --> PointRef
    WALEntry --> WALOperation
```

## Именованные типы

| Тип | Значения или назначение |
|---|---|
| `DistanceMetric` | `COSINE`, `DOT`, `L2` |
| `SearchMode` | `EXACT`, `APPROXIMATE` |
| `FilterOperator` | `EQ`, `GT`, `GTE`, `LT`, `LTE` |
| `ErrorCode` | стабильные машинные коды ошибок БД |
| `WALOperationType` | тип операции, записанной в WAL |
| `LSN` | монотонный номер записи WAL |
| `ShardID` | идентификатор шарда |

## Внутренние порты

```mermaid
classDiagram
    class PointStore {
        <<interface>>
        +UpsertBatch(ctx, collection, points) error
        +Get(ctx, collection, ref) VectorPoint, error
        +DeleteBatch(ctx, collection, refs) error
        +Filter(ctx, collection, filter) PointRef[], error
        +Descendants(ctx, collection, root) PointRef[], error
    }
    class VectorIndex {
        <<interface>>
        +UpsertBatch(ctx, points) error
        +DeleteBatch(ctx, refs) error
        +Search(ctx, request) Neighbor[], error
    }
    class WAL {
        <<interface>>
        +Append(ctx, operation) LSN, error
        +Sync() error
        +ReadAfter(ctx, lsn, limit) WALEntry[], error
        +TruncateThrough(ctx, lsn) error
        +Close() error
    }
    class SnapshotStore {
        <<interface>>
        +Save(ctx, snapshot) error
        +LoadLatest(ctx) Snapshot, error
    }
    class ShardRouter {
        <<interface>>
        +ShardFor(partitionKey) ShardID, error
        +AllShards() ShardID[]
    }
    class Replicator {
        <<interface>>
        +Start(ctx) error
        +Stop(ctx) error
    }
    VectorDB ..> PointStore
    VectorDB ..> VectorIndex
    VectorDB ..> WAL
    VectorDB ..> SnapshotStore
    VectorDB ..> ShardRouter
    WAL ..> Replicator
```

## Важные правила

- Корень использует `PartitionKey = ID`; его потомки передают тот же `PartitionKey`.
- Пустой `SearchMode` трактуется сервисом как `EXACT`.
- ЛР‑1 фиксирует контракты, но не реализует алгоритмы.
