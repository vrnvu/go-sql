package workerpool

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/vrnvu/go-sql/internal/client"
	"github.com/vrnvu/go-sql/internal/query"
	"pgregory.net/rapid"
)

// testQueryReader is a test implementation of the query.Reader interface
// it will successfully read until maxCalls is reached
type testQueryReader struct {
	calls    int
	maxCalls int
}

func (t *testQueryReader) Next() (query.Query, bool, error) {
	defer func() {
		t.calls++
	}()

	if t.calls >= t.maxCalls {
		return query.Query{}, false, nil
	}

	testQuery, err := testQuery(t.calls)
	if err != nil {
		return query.Query{}, false, err
	}

	return *testQuery, true, nil
}

type testDeterministicClient struct{}

func (t *testDeterministicClient) Ping(_ context.Context) error {
	return nil
}

func (t *testDeterministicClient) Query(_ context.Context, _ string) (*client.Response, error) {
	return &client.Response{Duration: 1 * time.Second}, nil
}

func testQuery(i int) (*query.Query, error) {
	hostname := fmt.Sprintf("hostname-%d", i)
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 1, 0, 0, 1, 0, time.UTC)
	return &query.Query{Hostname: hostname, StartTime: startTime, EndTime: endTime}, nil
}

func TestNewProperties(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		numWorkers := rapid.IntRange(1, MaxWorkers).Draw(t, "numWorkers")
		wp, err := New(numWorkers, &testDeterministicClient{}, &testQueryReader{maxCalls: 10})
		assert.NoError(t, err)
		assert.NotNil(t, wp)
	})
}

func TestNewZeroWorkers(t *testing.T) {
	t.Parallel()
	wp, err := New(0, &testDeterministicClient{}, &testQueryReader{maxCalls: 10})
	assert.Error(t, err)
	assert.Nil(t, wp)
	snaps.MatchSnapshot(t, err.Error())
}

func TestNewTooManyWorkers(t *testing.T) {
	t.Parallel()
	wp, err := New(MaxWorkers+1, &testDeterministicClient{}, &testQueryReader{maxCalls: 10})
	assert.Error(t, err)
	assert.Nil(t, wp)
	snaps.MatchSnapshot(t, err.Error())
}

func TestWorkerPoolIsCancel(t *testing.T) {
	t.Parallel()
	wp, err := New(1, &testDeterministicClient{}, &testQueryReader{maxCalls: 10})
	assert.NoError(t, err)
	assert.NotNil(t, wp)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	metrics, err := wp.Run(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	snaps.MatchSnapshot(t, metrics.Table())
}

// This is a cool property test (imo)
// We can prove our workerpool is deterministic in the number of workers
func TestSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: workerpool snapshot")
	}
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		numWorkers := rapid.IntRange(1, MaxWorkers).Draw(t, "numWorkers")
		wp, err := New(numWorkers, &testDeterministicClient{}, &testQueryReader{maxCalls: 10})
		assert.NoError(t, err)
		assert.NotNil(t, wp)

		ctx := t.Context()
		metrics, err := wp.Run(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
		snaps.MatchSnapshot(t, metrics.Table())
	})
}

// Another cool test, for any worker, rows and host, test worker pool processes everything
func TestWorkerPoolProcessesAllQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: workerpool snapshot")
	}
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		numWorkers := rapid.IntRange(1, MaxWorkers).Draw(t, "numWorkers")
		numRows := rapid.IntRange(1_000, 10_000).Draw(t, "numRows")
		numHosts := rapid.IntRange(1, 50).Draw(t, "numHosts")

		csvContent := "hostname,start_time,end_time\n"
		for range numRows {
			hostID := rand.Intn(numHosts) + 1 //nolint:gosec
			csvContent += fmt.Sprintf("host%d,2023-01-01 10:00:00,2023-01-01 10:01:00\n", hostID)
		}

		reader := strings.NewReader(csvContent)
		csvReader := csv.NewReader(reader)
		queryReader, err := query.NewQueryReader(csvReader)
		assert.NoError(t, err)
		assert.NotNil(t, queryReader)

		client := &testDeterministicClient{}
		wp, err := New(numWorkers, client, queryReader)
		assert.NotNil(t, wp)
		assert.NoError(t, err)

		ctx := t.Context()
		metrics, err := wp.Run(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)

		assert.Equal(t, numRows, metrics.NumberOfQueries)
		assert.Equal(t, 0, metrics.SkippedQueries)
		assert.Equal(t, 0, metrics.FailedQueries)
	})
}

func TestWorkerPoolMapsHostnameToWorker(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: workerpool snapshot")
	}
	t.Parallel()
	wp, err := New(2, &testDeterministicClient{}, &testQueryReader{maxCalls: 10})
	assert.NoError(t, err)
	assert.NotNil(t, wp)

	hostname1 := "host1"
	hostname2 := "host2"
	worker1 := wp.getWorker(hostname1)
	rapid.Check(t, func(t *rapid.T) {
		numWorkers := rapid.IntRange(1, MaxWorkers).Draw(t, "numWorkers")
		for range numWorkers {
			worker2 := wp.getWorker(hostname1)
			assert.Equal(t, worker1, worker2)
		}
	})

	worker2 := wp.getWorker(hostname2)
	assert.NotEqual(t, worker1, worker2)
}
