package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestCompareSimpleAndReservoirWhenInSampleSize(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		// Since we are in the sample size, we expect the results to be the same
		sampleSize := rapid.IntRange(1, SimpleMaxCapacity).Draw(t, "sampleSize")
		assert.True(t, sampleSize <= SimpleMaxCapacity)
		assert.True(t, ReservoirDefaultSampleSize <= SimpleMaxCapacity)

		simpleMetrics, err := NewSimpleWithCapacity(sampleSize)
		assert.NoError(t, err)

		reservoirMetrics, err := NewReservoirWithSize(sampleSize)
		assert.NoError(t, err)

		for range sampleSize {
			simpleMetrics.AddResponse(1 * time.Second)
			reservoirMetrics.AddResponse(1 * time.Second)
		}

		assert.Equal(t, simpleMetrics.Aggregate(), reservoirMetrics.Aggregate())
	})
}

// This test can be used to generate snapshots and smoke tests
func TestCompareSimpleAndReservoirWhenSampleSizeIsGreaterThanReservoirSampleSize(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		// We start at a number big enough so we trigger the sampling algorithm
		capacity := rapid.IntRange(10_000, SimpleMaxCapacity).Draw(t, "capacity")
		simpleMetrics, err := NewSimpleWithCapacity(capacity)
		assert.NoError(t, err)
		assert.NotNil(t, simpleMetrics)

		// Reservoir sample size is half of the sample size so we trigger the sampling algorithm
		reservoirSampleSize := capacity / 2
		reservoirMetrics, err := NewReservoirWithSize(reservoirSampleSize)
		assert.NoError(t, err)
		assert.NotNil(t, reservoirMetrics)

		for range capacity {
			duration := time.Duration(rapid.IntRange(1, 1_000).Draw(t, "duration")) * time.Second
			simpleMetrics.AddResponse(duration)
			reservoirMetrics.AddResponse(duration)
		}

		simpleResult := simpleMetrics.Aggregate()
		reservoirResult := reservoirMetrics.Aggregate()

		assert.Equal(t, simpleResult.NumberOfQueries, reservoirResult.NumberOfQueries)
		assert.Equal(t, simpleResult.TotalProcessingTime, reservoirResult.TotalProcessingTime)
		assert.Equal(t, simpleResult.MinResponse, reservoirResult.MinResponse)
		assert.Equal(t, simpleResult.AverageResponse, reservoirResult.AverageResponse)
		assert.Equal(t, simpleResult.MaxResponse, reservoirResult.MaxResponse)

		// Sometimes the difference is due to the sampling algorithm
		// assert.NotEqual(t, simpleResult.MedianResponse, reservoirResult.MedianResponse)
		// assert.Equal(t, simpleResult.MedianResponse, reservoirResult.MedianResponse)
	})
}
