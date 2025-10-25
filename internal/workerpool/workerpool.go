package workerpool

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/vrnvu/go-sql/internal/client"
	"github.com/vrnvu/go-sql/internal/metrics"
	"github.com/vrnvu/go-sql/internal/query"
)

const (
	// MaxWorkers is the maximum number of parallel workers querying TigerData
	MaxWorkers = 1024
)

// Result is a single query result, containing the worker ID, hostname, request start time, and request end time
// Note: Simple representation, state can be Skipped, Failed or Successful(duration)
type Result struct {
	skipped  bool
	failed   bool
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
	client client.Client

	queryReader         query.Reader
	queries             []chan query.Query
	results             chan Result
	mapHostnameToWorker map[string]chan query.Query
	lastWorkerIdx       int
	numWorkers          int
	wgWorkers           sync.WaitGroup

	wgMetrics     sync.WaitGroup
	simpleMetrics *metrics.Simple
}

// New creates a new WorkerPool with the given number of workers
func New(numWorkers int, client client.Client, queryReader query.Reader) (*WorkerPool, error) {
	if numWorkers < 1 {
		return nil, fmt.Errorf("number of workers must be greater than 0")
	}

	if numWorkers > MaxWorkers {
		return nil, fmt.Errorf("number of workers must be less than %d", MaxWorkers)
	}

	queries := make([]chan query.Query, numWorkers)
	for i := range numWorkers {
		queries[i] = make(chan query.Query)
	}

	// TODO probably we can simplify the interface and ping and smoke test in the constructor
	if client.Ping() != nil {
		return nil, fmt.Errorf("failed to ping client")
	}

	return &WorkerPool{
		queryReader:         queryReader,
		client:              client,
		simpleMetrics:       metrics.NewSimple(),
		queries:             queries,
		results:             make(chan Result),
		mapHostnameToWorker: make(map[string]chan query.Query),
		numWorkers:          numWorkers,
	}, nil
}

// Run reads queries from the query reader and distributes them to the workers
// It collects metrics from the results channel and returns the aggregated metrics
// Returns error in panics and context cancellation
// 1. it starts all the workers (numWorkers) and the metrics collector (1)
// 2. it reads queries from the query reader and distributes them to the workers
// 3. it waits for all the workers to finish and closes the results channel
// 4. it waits for the metrics collector to finish and returns the aggregated metrics
func (wp *WorkerPool) Run(ctx context.Context) (metrics.Result, error) {
	wp.wgWorkers.Add(wp.numWorkers)
	for i := 0; i < wp.numWorkers; i++ {
		go wp.worker(ctx, wp.queries[i])
	}

	wp.wgMetrics.Add(1)
	go wp.CollectMetrics()

	for {
		select {
		case <-ctx.Done():
			return metrics.Result{}, ctx.Err()
		default:
		}

		query, hasMore, err := wp.queryReader.Next()
		if !hasMore {
			break
		}
		if err != nil {
			log.Printf("warning: skipped reading query due to error: %v", err)
			wp.sendSkipped(ctx)
			continue
		}
		if err := wp.sendQuery(ctx, query); err != nil {
			return metrics.Result{}, err
		}
	}

	for i := 0; i < wp.numWorkers; i++ {
		close(wp.queries[i])
	}
	wp.wgWorkers.Wait()
	close(wp.results)
	wp.wgMetrics.Wait()

	return wp.simpleMetrics.Aggregate(), nil
}

func (wp *WorkerPool) sendQuery(ctx context.Context, query query.Query) error {
	queryChan := wp.getWorker(query.Hostname)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case queryChan <- query:
	}
	return nil
}

func (wp *WorkerPool) sendResult(ctx context.Context, result Result) {
	select {
	case <-ctx.Done():
		return
	case wp.results <- result:
	}
}

func (wp *WorkerPool) sendSkipped(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case wp.results <- Result{skipped: true}:
	}
}

func (wp *WorkerPool) sendFailed(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case wp.results <- Result{failed: true}:
	}
}

func (wp *WorkerPool) worker(ctx context.Context, queries <-chan query.Query) {
	defer wp.wgWorkers.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case query, ok := <-queries:
			if !ok {
				return
			}

			response, err := wp.client.Query(ctx, query.Build())
			if err != nil {
				log.Printf("worker: failed query: %v", err)
				wp.sendFailed(ctx)
				continue
			}

			wp.sendResult(ctx, Result{Duration: response.Duration})
		}
	}
}

// getWorker returns the query channel for the given hostname
// If the hostname is not mapped, it uses round robin
func (wp *WorkerPool) getWorker(hostname string) chan query.Query {
	if queryChan, exists := wp.mapHostnameToWorker[hostname]; exists {
		return queryChan
	}

	queryChan := wp.queries[wp.lastWorkerIdx]
	wp.mapHostnameToWorker[hostname] = queryChan
	wp.lastWorkerIdx = (wp.lastWorkerIdx + 1) % wp.numWorkers
	return queryChan
}

// CollectMetrics collects results from the results channel and updates metrics
// Since this is unbounded, acts a sync mechanism, we could have multiple metrics collectors
// Then we would need to make our metrics thread safe
func (wp *WorkerPool) CollectMetrics() {
	defer wp.wgMetrics.Done()
	for result := range wp.results {
		if result.skipped {
			wp.simpleMetrics.AddSkipped()
		} else if result.failed {
			wp.simpleMetrics.AddFailed()
		} else {
			wp.simpleMetrics.AddResponse(result.Duration)
		}
	}
}
