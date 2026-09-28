package model_test

import (
	"errors"
	"testing"

	"vectordb/internal/model"
)

func TestDBErrorImplementsError(t *testing.T) {
	var _ error = model.DBError{}

	err := model.DBError{
		Code:    model.ErrorInvalidVector,
		Message: "vector contains NaN",
	}
	if got, want := err.Error(), "INVALID_VECTOR: vector contains NaN"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	var target model.DBError
	if !errors.As(err, &target) {
		t.Fatal("DBError must work with errors.As")
	}
}

func TestPointRefCanBeMapKey(t *testing.T) {
	ref := model.PointRef{ID: "chunk-1", PartitionKey: "document-1"}
	refs := map[model.PointRef]struct{}{ref: {}}
	if _, ok := refs[ref]; !ok {
		t.Fatal("PointRef must be comparable")
	}
}

func TestPointUpsertOptionalFieldsMayBeNil(t *testing.T) {
	point := model.PointUpsert{ID: "chunk-1", PartitionKey: "document-1"}
	if point.ParentID != nil || point.Vector != nil || point.Content != nil {
		t.Fatal("omitted fields must stay nil")
	}
}
