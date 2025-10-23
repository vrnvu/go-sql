package workerpool

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/vrnvu/go-sql/internal/query"
	"pgregory.net/rapid"
)

func Query(i int) (*query.Query, error) {
	hostname := fmt.Sprintf("hostname-%d", i)
	startTime, err := time.Parse(time.DateTime, "2025-01-01 00:00:00")
	if err != nil {
		return nil, err
	}
	endTime, err := time.Parse(time.DateTime, "2025-01-01 00:00:01")
	if err != nil {
		return nil, err
	}
	return &query.Query{Hostname: hostname, StartTime: startTime, EndTime: endTime}, nil
}

func TestNewProperties(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		numWorkers := rapid.IntRange(1, runtime.NumCPU()).Draw(t, "numWorkers")
		wp, err := New(numWorkers)
		assert.NoError(t, err)
		assert.NotNil(t, wp)
	})
}

func TestNewZeroWorkers(t *testing.T) {
	t.Parallel()
	wp, err := New(0)
	assert.Error(t, err)
	assert.Nil(t, wp)
	snaps.MatchSnapshot(t, err.Error())
}

func TestNewTooManyWorkers(t *testing.T) {
	t.Parallel()
	wp, err := New(runtime.NumCPU() + 1)
	assert.Error(t, err)
	assert.Nil(t, wp)
	snaps.MatchSnapshot(t, err.Error())
}

func TestWorkerPoolIsCancel(t *testing.T) {
	t.Parallel()
	wp, err := New(1)
	assert.NoError(t, err)
	assert.NotNil(t, wp)

	ctx, cancel := context.WithCancel(t.Context())
	wp.RunWorkers(ctx)
	cancel()

	for _, c := range wp.queries {
		select {
		case <-c:
			assert.True(t, false, "channel is still open")
		default:
			assert.True(t, true, "channel is closed")
		}
	}
}

// TODO: we can prove our workerpool is deterministic in the number of workers
// But first we need to do DI in our worker dependencies
// func TestSnapshot(t *testing.T) {
// 	t.Parallel()
// 	rapid.Check(t, func(t *rapid.T) {
// 		numWorkers := rapid.IntRange(1, runtime.NumCPU()).Draw(t, "numWorkers")
// 		wp, err := New(numWorkers)
// 		assert.NoError(t, err)
// 		assert.NotNil(t, wp)

// 		ctx := t.Context()
// 		done := make(chan bool)

// 		wp.RunWorkers(ctx)
// 		go wp.SendMetrics(ctx, done)

// 		for i := range 10 {
// 			query, err := Query(i)
// 			assert.NoError(t, err)
// 			assert.NotNil(t, query)

// 			if err := wp.RunQuery(ctx, *query); err != nil {
// 				log.Fatalf("Error running query: %v", err)
// 			}
// 		}

// 		wp.Close()
// 		<-done

// 		metrics := wp.AggregateMetrics()
// 		snaps.MatchSnapshot(t, metrics.AverageResponse.Round(time.Millisecond))
// 		snaps.MatchSnapshot(t, metrics.MaxResponse.Round(time.Millisecond))
// 		snaps.MatchSnapshot(t, metrics.MedianResponse.Round(time.Millisecond))
// 		snaps.MatchSnapshot(t, metrics.MinResponse.Round(time.Millisecond))
// 		snaps.MatchSnapshot(t, metrics.NumberOfQueries)
// 		snaps.MatchSnapshot(t, metrics.TotalProcessingTime.Round(time.Millisecond))
// 	})
// }
