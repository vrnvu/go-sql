package metrics

import (
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestNewReservoir(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sampleSize := rapid.IntRange(1, ReservoirDefaultSampleSize).Draw(t, "sampleSize")
		metrics, err := NewReservoirWithSize(sampleSize)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
	})
}

func TestNewReservoirWithSizeZeroSize(t *testing.T) {
	metrics, err := NewReservoirWithSize(0)
	assert.Error(t, err)
	assert.Nil(t, metrics)
	snaps.MatchSnapshot(t, err.Error())
}

func TestNewReservoirWithSizeTooManySize(t *testing.T) {
	metrics, err := NewReservoirWithSize(ReservoirMaxCapacity + 1)
	assert.Error(t, err)
	assert.Nil(t, metrics)
	snaps.MatchSnapshot(t, err.Error())
}

func TestReservoirAddResponseDoesNotPanicWhenMaxCapacityIsReached(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, ReservoirMaxCapacity).Draw(t, "capacity")
		metrics, err := NewReservoirWithSize(capacity)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)

		for range capacity {
			metrics.AddResponse(time.Duration(1) * time.Second)
		}

		assert.NotPanics(t, func() {
			metrics.AddResponse(1 * time.Second)
		})
	})
}

func TestReservoirMetricsAggregate(t *testing.T) {
	metrics := NewReservoir()
	metrics.AddResponse(1 * time.Second)
	metrics.AddResponse(2 * time.Second)
	metrics.AddResponse(3 * time.Second)
	metrics.AddResponse(4 * time.Second)
	metrics.AddResponse(5 * time.Second)
	metrics.AddResponse(6 * time.Second)
	metrics.AddResponse(7 * time.Second)
	metrics.AddResponse(8 * time.Second)
	metrics.AddResponse(9 * time.Second)
	metrics.AddResponse(10 * time.Second)

	result := metrics.Aggregate()
	assert.Equal(t, 10, result.NumberOfQueries)
	assert.Equal(t, 55*time.Second, result.TotalProcessingTime)
	assert.Equal(t, 1*time.Second, result.MinResponse)
	assert.Equal(t, 6*time.Second, result.MedianResponse)
	assert.Equal(t, 55*time.Second/10, result.AverageResponse)
	assert.Equal(t, 10*time.Second, result.MaxResponse)
	snaps.MatchSnapshot(t, result)
}
