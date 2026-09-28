package model

type DistanceMetric string

const (
	MetricCosine DistanceMetric = "COSINE"
	MetricDot    DistanceMetric = "DOT"
	MetricL2     DistanceMetric = "L2"
)

type SearchMode string

const (
	SearchModeExact       SearchMode = "EXACT"
	SearchModeApproximate SearchMode = "APPROXIMATE"
)

type FilterOperator string

const (
	FilterEQ  FilterOperator = "EQ"
	FilterGT  FilterOperator = "GT"
	FilterGTE FilterOperator = "GTE"
	FilterLT  FilterOperator = "LT"
	FilterLTE FilterOperator = "LTE"
)

type WALOperationType string

const (
	WALCreateCollection WALOperationType = "CREATE_COLLECTION"
	WALDeleteCollection WALOperationType = "DELETE_COLLECTION"
	WALUpsertBatch      WALOperationType = "UPSERT_BATCH"
	WALDeleteBatch      WALOperationType = "DELETE_BATCH"
)

type LSN uint64

type ShardID uint32

type CollectionConfig struct {
	Name              string
	Dimension         int
	Metric            DistanceMetric
	ShardCount        int
	ReplicationFactor int
}

type PointRef struct {
	ID           string
	PartitionKey string
}

type VectorPoint struct {
	ID           string
	PartitionKey string
	ParentID     *string
	Vector       []float32
	Content      string
	Metadata     map[string]any
}

type PointUpsert struct {
	ID                 string
	PartitionKey       string
	ParentID           *string
	Vector             *[]float32
	Content            *string
	MetadataPatch      map[string]any
	DeleteMetadataKeys []string
}

type Filter struct {
	Conditions []Condition
}

type Condition struct {
	Field    string
	Operator FilterOperator
	Value    any
}

type SearchQuery struct {
	Vector        []float32
	TopK          int
	Mode          SearchMode
	Filter        *Filter
	IncludeVector bool
}

type SearchResult struct {
	Ref      PointRef
	ParentID *string
	Content  string
	Metadata map[string]any
	Score    float32
	Vector   []float32
}

type BatchFailure struct {
	Ref   PointRef
	Error DBError
}

type BatchResult struct {
	Applied []PointRef
	Failed  []BatchFailure
}

type Neighbor struct {
	Ref   PointRef
	Score float32
}

type IndexSearchRequest struct {
	Vector      []float32
	TopK        int
	Mode        SearchMode
	AllowedRefs map[PointRef]struct{}
}

type WALOperation struct {
	Type       WALOperationType
	Collection string
	Payload    []byte
}

type WALEntry struct {
	LSN       LSN
	Operation WALOperation
}

type Snapshot struct {
	LastLSN LSN
	Payload []byte
}
