package workerpool

import (
	"runtime"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestNewProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numWorkers := rapid.IntRange(1, runtime.NumCPU()).Draw(t, "numWorkers")
		wp, err := New(numWorkers)
		assert.NoError(t, err)
		assert.NotNil(t, wp)
	})
}

func TestNewZeroWorkers(t *testing.T) {
	wp, err := New(0)
	assert.Error(t, err)
	assert.Nil(t, wp)
	snaps.MatchSnapshot(t, err.Error())
}

func TestNewTooManyWorkers(t *testing.T) {
	wp, err := New(runtime.NumCPU() + 1)
	assert.Error(t, err)
	assert.Nil(t, wp)
	snaps.MatchSnapshot(t, err.Error())
}
