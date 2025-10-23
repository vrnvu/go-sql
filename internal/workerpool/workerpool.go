package workerpool

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
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
	workerWg            sync.WaitGroup
	numWorkers          int
	mapHostnameToWorker map[string]chan query.Query
	lastWorkerIdx       int
	results             chan Result
	queries             []chan query.Query
	simpleMetrics       *metrics.Simple
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
		numWorkers:          numWorkers,
		mapHostnameToWorker: make(map[string]chan query.Query),
		lastWorkerIdx:       0,
		results:             make(chan Result),
		simpleMetrics:       metrics.NewSimple(),
		queries:             queries,
	}, nil
}

// RunWorkers starts the workers in the WorkerPool
func (wp *WorkerPool) RunWorkers(ctx context.Context) {
	wp.workerWg.Add(wp.numWorkers)
	for i := 0; i < wp.numWorkers; i++ {
		go worker(ctx, i, wp.queries[wp.lastWorkerIdx], wp.results, &wp.workerWg)
		wp.lastWorkerIdx = (wp.lastWorkerIdx + 1) % wp.numWorkers
	}
}

// RunQuery allocates a worker to run a query on the WorkerPool
// Thread safety: `lastWorkerIdx` is not thread safe
func (wp *WorkerPool) RunQuery(ctx context.Context, query query.Query) error {
	if queryWorker, ok := wp.mapHostnameToWorker[query.Hostname]; ok {
		return runQuery(ctx, queryWorker, query)
	}

	queryWorker := wp.queries[wp.lastWorkerIdx]
	wp.lastWorkerIdx = (wp.lastWorkerIdx + 1) % wp.numWorkers
	wp.mapHostnameToWorker[query.Hostname] = queryWorker
	return runQuery(ctx, queryWorker, query)
}

func runQuery(ctx context.Context, queryWorker chan query.Query, query query.Query) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case queryWorker <- query:
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
// Thread safety: waits for all workers to finish before closing channels
func (wp *WorkerPool) Close() {
	// Close query channels to signal workers to stop
	for _, queryWorker := range wp.queries {
		close(queryWorker)
	}

	// Wait for all workers to finish
	wp.workerWg.Wait()

	// Now it's safe to close the results channel
	close(wp.results)
}

func worker(ctx context.Context, id int, queries <-chan query.Query, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case query, ok := <-queries:
			if !ok {
				return
			}
			start := time.Now()
			// TODO simulate query execution
			fmt.Printf("worker: id-%d executing query: hostname-%s start_time-%s end_time-%s\n", id, query.Hostname, query.StartTime, query.EndTime)
			time.Sleep(2 * time.Millisecond)
			end := time.Now()

			select {
			case results <- Result{Duration: end.Sub(start)}:
			case <-ctx.Done():
				log.Panicf("worker: id-%d context cancelled, not sending result\n", id)
				return
			}
		}
	}
}
