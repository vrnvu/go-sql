package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestNewSimpleWithCapacityProperties(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, SimpleMaxCapacity).Draw(t, "capacity")
		metrics, err := NewSimpleWithCapacity(capacity)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Equal(t, 0, len(metrics.responses))
		assert.Equal(t, capacity, cap(metrics.responses))
	})
}

func TestNewSimpleWithCapacityZeroCapacity(t *testing.T) {
	t.Parallel()
	metrics, err := NewSimpleWithCapacity(0)
	assert.Error(t, err)
	assert.Nil(t, metrics)
	snaps.MatchSnapshot(t, err.Error())
}

func TestNewSimpleWithCapacityTooManyCapacity(t *testing.T) {
	t.Parallel()
	metrics, err := NewSimpleWithCapacity(SimpleMaxCapacity + 1)
	assert.Error(t, err)
	assert.Nil(t, metrics)
	snaps.MatchSnapshot(t, err.Error())
}

func TestSimpleAddResponsePanicsWhenMaxCapacityIsReached(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: simple add response panics test")
	}
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, SimpleMaxCapacity).Draw(t, "capacity")
		metrics, err := NewSimpleWithCapacity(capacity)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)

		for range capacity {
			metrics.AddResponse(time.Duration(1) * time.Second)
		}

		assert.Panics(t, func() {
			metrics.AddResponse(1 * time.Second)
		})
	})
}

func TestSimpleMetricsAggregate(t *testing.T) {
	t.Parallel()
	metrics := NewSimple()
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

func TestSimpleAddSkippedAndFailedToMaxThenOverflow(t *testing.T) {
	t.Parallel()
	metrics := NewSimple()
	metrics.skippedQueries = math.MaxInt64
	metrics.failedQueries = math.MaxInt64

	assert.Panics(t, func() {
		metrics.AddSkipped()
	})
	assert.Panics(t, func() {
		metrics.AddFailed()
	})
}
