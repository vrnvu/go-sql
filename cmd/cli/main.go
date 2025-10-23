package main

import (
	"context"
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/vrnvu/go-sql/internal/query"
	"github.com/vrnvu/go-sql/internal/workerpool"
)

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

	// TODO Since this is IO bound to Network, I'd consider a timeout
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp, err := workerpool.New(numWorkers)
	if err != nil {
		log.Fatalf("error creating worker pool: %v", err)
	}

	wp.RunWorkers(ctx)
	done := make(chan bool)
	go wp.CollectResults(ctx, done)

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
		query := query.Query{
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
