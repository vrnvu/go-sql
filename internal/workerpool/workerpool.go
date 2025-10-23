package workerpool

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/vrnvu/go-sql/internal/metrics"
	"github.com/vrnvu/go-sql/internal/query"
)

// Result is a single query result, containing the worker ID, hostname, request start time, and request end time
type Result struct {
	Duration time.Duration
}

// WorkerPool is a pool of workers that can execute queries
// It maintains a map of hostnames to query channels, and a round-robin index for selecting the next worker
// Strategy: fixed hostname to worker, or else round robin
// For example:
// For example: 4 cores, 4 workers, 4 query channels
// We Map worker to Query channels:
// queries = [chan string, chan string, chan string, chan string]
// - worker 0: query channel 0
// - worker 1: query channel 1
// - worker 2: query channel 2
// - worker 3: query channel 3
// When we receive a query, we map the hostname to the corresponding query channel or else round robin
// Example:
// query.hostname = "host1" -> query channel 0
// query.hostname = "host3" -> query channel 2
// query.hostname = "host2" -> query channel 1
// query.hostname = "host1" -> query channel 0
// query.hostname = "host2" -> query channel 1
// query.hostname = "host3" -> query channel 2
// query.hostname = "host4" -> query channel 3
// query.hostname = "host5" -> query channel 0 // idx % numWorkers
type WorkerPool struct {
	numWorkers             int
	map_hostname_to_worker map[string]chan query.Query
	last_worker_idx        int
	results                chan Result
	simpleMetrics          *metrics.Simple
	queries                []chan query.Query
}

// New creates a new WorkerPool with the given number of workers
func New(numWorkers int) (*WorkerPool, error) {
	if numWorkers < 1 {
		return nil, fmt.Errorf("number of workers must be greater than 0")
	}

	if numWorkers > runtime.NumCPU() {
		return nil, fmt.Errorf("number of workers must be less than the number of CPUs")
	}

	queries := make([]chan query.Query, numWorkers)
	for i := range numWorkers {
		queries[i] = make(chan query.Query)
	}

	return &WorkerPool{
		numWorkers:             numWorkers,
		map_hostname_to_worker: make(map[string]chan query.Query),
		last_worker_idx:        0,
		results:                make(chan Result),
		simpleMetrics:          metrics.NewSimple(),
		queries:                queries,
	}, nil
}

// RunWorkers starts the workers in the WorkerPool
func (wp *WorkerPool) RunWorkers(ctx context.Context) {
	for i := 0; i < wp.numWorkers; i++ {
		go worker(ctx, i, wp.queries[wp.last_worker_idx], wp.results)
		wp.last_worker_idx = (wp.last_worker_idx + 1) % wp.numWorkers
	}
}

// RunQuery allocates a worker to run a query on the WorkerPool
// Thread safety: `last_worker_idx` is not thread safe
func (wp *WorkerPool) RunQuery(ctx context.Context, query query.Query) error {
	if query_worker, ok := wp.map_hostname_to_worker[query.Hostname]; ok {
		return runQuery(ctx, query_worker, query)
	}

	query_worker := wp.queries[wp.last_worker_idx]
	wp.last_worker_idx = (wp.last_worker_idx + 1) % wp.numWorkers
	wp.map_hostname_to_worker[query.Hostname] = query_worker
	return runQuery(ctx, query_worker, query)
}

func runQuery(ctx context.Context, query_worker chan query.Query, query query.Query) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case query_worker <- query:
		return nil
	}
}

// SendMetrics sends the results from the workers to our metrics aggregator
func (wp *WorkerPool) SendMetrics(ctx context.Context, done chan<- bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-wp.results:
			if !ok {
				done <- true
				return
			}
			wp.simpleMetrics.AddResponse(result.Duration)
		}
	}
}

// AggregateMetrics aggregates the metrics from the workers
func (wp *WorkerPool) AggregateMetrics() metrics.Result {
	return wp.simpleMetrics.Aggregate()
}

// Close all the query channels and the results channel in order to terminate the workers
// Thread safety: all the channels are closed in a single goroutine, so there is no race condition
func (wp *WorkerPool) Close() {
	for _, query_worker := range wp.queries {
		close(query_worker)
	}

	close(wp.results)
}

func worker(ctx context.Context, id int, queries <-chan query.Query, results chan<- Result) {
	for {
		select {
		case <-ctx.Done():
			return
		case query := <-queries:
			start := time.Now()
			// TODO simulate query execution
			fmt.Printf("worker: id-%d executing query: hostname-%s start_time-%s end_time-%s\n", id, query.Hostname, query.StartTime, query.EndTime)
			time.Sleep(2 * time.Millisecond)
			end := time.Now()
			results <- Result{
				Duration: end.Sub(start),
			}
		}
	}
}
