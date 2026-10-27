package index

import (
	"math"
	"vectordb/internal/model"
)

func distance(query, vector []float32, metric model.DistanceMetric) float64 {
	var dot, queryNorm, vectorNorm, squaredDistance float64
	for i, value := range query {
		a, b := float64(value), float64(vector[i])
		dot += a * b
		queryNorm += a * a
		vectorNorm += b * b
		delta := a - b
		squaredDistance += delta * delta
	}
	switch metric {
	case model.MetricCosine:
		return dot / (math.Sqrt(queryNorm) * math.Sqrt(vectorNorm))
	case model.MetricDot:
		return dot
	default:
		return math.Sqrt(squaredDistance)
	}
}
