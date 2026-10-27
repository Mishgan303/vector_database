package model

type ErrorCode string

const (
	ErrorInvalidArgument         ErrorCode = "INVALID_ARGUMENT"
	ErrorCollectionNotFound      ErrorCode = "COLLECTION_NOT_FOUND"
	ErrorCollectionAlreadyExists ErrorCode = "COLLECTION_ALREADY_EXISTS"
	ErrorPointNotFound           ErrorCode = "POINT_NOT_FOUND"
	ErrorParentNotFound          ErrorCode = "PARENT_NOT_FOUND"
	ErrorParentPartitionMismatch ErrorCode = "PARENT_PARTITION_MISMATCH"
	ErrorDimensionMismatch       ErrorCode = "DIMENSION_MISMATCH"
	ErrorInvalidCollectionConfig ErrorCode = "INVALID_COLLECTION_CONFIG"
	ErrorInvalidVector           ErrorCode = "INVALID_VECTOR"
	ErrorInvalidMetadata         ErrorCode = "INVALID_METADATA"
	ErrorInvalidFilter           ErrorCode = "INVALID_FILTER"
	ErrorPartitionKeyImmutable   ErrorCode = "PARTITION_KEY_IMMUTABLE"
	ErrorUnsupportedSearchMode   ErrorCode = "UNSUPPORTED_SEARCH_MODE"
	ErrorPartialFailure          ErrorCode = "PARTIAL_FAILURE"
	ErrorStorage                 ErrorCode = "STORAGE_ERROR"
)

type DBError struct {
	Code    ErrorCode
	Message string
	Details map[string]any
}

func (e DBError) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Message
}
