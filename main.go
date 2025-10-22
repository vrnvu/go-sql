package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"
)

type Query struct {
	Hostname  string    `csv:"hostname"`
	StartTime time.Time `csv:"start_time"`
	EndTime   time.Time `csv:"end_time"`
}

// TODO
func (q *Query) Build() string {
	return fmt.Sprintf("SELECT * FROM cpu_usage WHERE hostname = %s AND ts BETWEEN %s AND %s", q.Hostname, q.StartTime, q.EndTime)
}

type Result struct {
	WorkerID         int
	Hostname         string
	RequestStartTime time.Time
	RequestEndTime   time.Time
	// TODO can we get time to first byte?
	// ResponseStartTime time.Time
	// ResponseEndTime   time.Time
}

type WorkerPool struct {
	numWorkers             int
	map_hostname_to_worker map[string]chan Query
	last_worker_idx        int
	results                chan Result
	queries                []chan Query
}

func NewWorkerPool(numWorkers int) *WorkerPool {
	queries := make([]chan Query, numWorkers)
	for i := range numWorkers {
		queries[i] = make(chan Query)
	}

	return &WorkerPool{
		numWorkers:             numWorkers,
		map_hostname_to_worker: make(map[string]chan Query),
		last_worker_idx:        0,
		results:                make(chan Result),
		queries:                queries,
	}
}

func (wp *WorkerPool) RunWorkers(ctx context.Context) {
	for i := 0; i < wp.numWorkers; i++ {
		go worker(ctx, i, wp.queries[wp.last_worker_idx], wp.results)
		wp.last_worker_idx = (wp.last_worker_idx + 1) % wp.numWorkers
	}
}

// TODO doc thread safety: last_worker_idx is not thread safe
// RunQuery must be run sequentially
func (wp *WorkerPool) RunQuery(ctx context.Context, query Query) error {
	if query_worker, ok := wp.map_hostname_to_worker[query.Hostname]; ok {
		return runQuery(ctx, query_worker, query)
	}

	query_worker := wp.queries[wp.last_worker_idx]
	wp.last_worker_idx = (wp.last_worker_idx + 1) % wp.numWorkers
	wp.map_hostname_to_worker[query.Hostname] = query_worker
	return runQuery(ctx, query_worker, query)
}

func runQuery(ctx context.Context, query_worker chan Query, query Query) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case query_worker <- query:
		return nil
	}
}

// TODO aggregate results
// # of queries processed,
// total processing time across all queries,
// the minimum query time (for a single query),
// the median query time,
// the average query time,
// and the maximum query time.
func (wp *WorkerPool) RunResults(ctx context.Context, done chan<- bool) {
	num_queries := 0
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-wp.results:
			if !ok {
				done <- true
				fmt.Printf("num_queries: %d\n", num_queries)
				return
			}
			num_queries++
			fmt.Println(result)
		}
	}
}

func (wp *WorkerPool) Close() {
	for _, query_worker := range wp.queries {
		close(query_worker)
	}

	close(wp.results)
}

func worker(ctx context.Context, id int, queries <-chan Query, results chan<- Result) {
	for {
		select {
		case <-ctx.Done():
			return
		case query := <-queries:
			start := time.Now()
			time.Sleep(2 * time.Millisecond)
			end := time.Now()
			results <- Result{
				WorkerID:         id,
				Hostname:         query.Hostname,
				RequestStartTime: start,
				RequestEndTime:   end,
			}
		}
	}
}

func main() {
	var inputPath string
	var numWorkers int

	flag.StringVar(&inputPath, "input", "", "Path to input CSV (defaults to stdin)")
	flag.IntVar(&numWorkers, "workers", 0, "Number of workers to use (defaults to number of cores)")
	flag.Parse()

	var reader *csv.Reader
	if inputPath == "" {
		log.Println("input path is empty, reading from stdin")
		reader = csv.NewReader(os.Stdin)
	} else {
		file, err := os.Open(inputPath)
		if err != nil {
			log.Fatalf("error opening input file: %v", err)
		}
		defer file.Close()
		reader = csv.NewReader(file)
	}

	numCores := runtime.NumCPU()
	if numWorkers < 1 || numCores < numWorkers {
		flag.Usage()
		log.Fatalf("number of workers must be greater than 0 and less than the number of cores: %d", numCores)
	}

	fields, err := reader.Read()
	if err != nil {
		log.Fatalf("error reading fields: %v", err)
	}
	if len(fields) != 3 {
		log.Fatalf("expected 3 fields, got %d", len(fields))
	}
	if fields[0] != "hostname" || fields[1] != "start_time" || fields[2] != "end_time" {
		log.Fatalf("expected fields to be hostname, start_time, end_time, got %v", fields)
	}

	// Worker pool, that reads queries channel
	// Loadbalancing strategy is: fixed hostname to worker, or else round robin
	// 4 cores, 4 workers, 4 query channels
	// queries = [chan string, chan string, chan string, chan string]
	// worker 0: query channel 0
	// worker 1: query channel 1
	// worker 2: query channel 2
	// worker 3: query channel 3
	// Example:
	// query.hostname = "host1" -> query channel 0
	// query.hostname = "host3" -> query channel 2
	// query.hostname = "host2" -> query channel 1
	// query.hostname = "host1" -> query channel 0
	// query.hostname = "host2" -> query channel 1
	// query.hostname = "host3" -> query channel 2
	// query.hostname = "host4" -> query channel 3
	// query.hostname = "host5" -> query channel 0 // idx % numWorkers

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := NewWorkerPool(numWorkers)
	wp.RunWorkers(ctx)

	done := make(chan bool)
	go wp.RunResults(ctx, done)

	for {
		select {
		case <-ctx.Done():
			log.Fatalf("Context cancelled: %v", ctx.Err())
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error reading input CSV: %v", err)
		}

		hostname := record[0]
		// TODO test context cancellation
		// if hostname == "host_000002" {
		// 	cancel()
		// }

		startTime, err := time.Parse(time.DateTime, record[1])
		if err != nil {
			log.Fatalf("error parsing start time: %v", err)
		}
		endTime, err := time.Parse(time.DateTime, record[2])
		if err != nil {
			log.Fatalf("error parsing end time: %v", err)
		}

		// TODO sync.Pool?
		query := Query{
			Hostname:  hostname,
			StartTime: startTime,
			EndTime:   endTime,
		}

		if err := wp.RunQuery(ctx, query); err != nil {
			log.Fatalf("Error running query: %v", err)
		}
	}

	wp.Close()
	<-done
}
