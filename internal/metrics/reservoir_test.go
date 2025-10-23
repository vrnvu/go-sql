package metrics

import (
	"testing"

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
