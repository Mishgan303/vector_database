package model

import (
	"maps"
	"math"
)

// Number normalizes the supported Go numeric types to finite float64 values.
func Number(value any) (float64, bool) {
	var number float64
	switch v := value.(type) {
	case int:
		number = float64(v)
	case int8:
		number = float64(v)
	case int16:
		number = float64(v)
	case int32:
		number = float64(v)
	case int64:
		number = float64(v)
	case uint:
		number = float64(v)
	case uint8:
		number = float64(v)
	case uint16:
		number = float64(v)
	case uint32:
		number = float64(v)
	case uint64:
		number = float64(v)
	case float32:
		number = float64(v)
	case float64:
		number = v
	default:
		return 0, false
	}
	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}

func ClonePoint(point VectorPoint) VectorPoint {
	point.Vector = append([]float32(nil), point.Vector...)
	if point.ParentID != nil {
		parent := *point.ParentID
		point.ParentID = &parent
	}
	point.Metadata = maps.Clone(point.Metadata)
	return point
}

func LessRef(a, b PointRef) bool {
	if a.PartitionKey != b.PartitionKey {
		return a.PartitionKey < b.PartitionKey
	}
	return a.ID < b.ID
}

func ValidateVector(vector []float32, dimension int) error {
	if len(vector) != dimension {
		return DBError{Code: ErrorDimensionMismatch, Message: "vector length differs from collection dimension"}
	}
	nonzero := false
	for _, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return DBError{Code: ErrorInvalidVector, Message: "vector must contain finite values"}
		}
		nonzero = nonzero || value != 0
	}
	if !nonzero {
		return DBError{Code: ErrorInvalidVector, Message: "vector must not be zero"}
	}
	return nil
}
