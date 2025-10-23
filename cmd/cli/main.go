package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
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
	var timeoutSeconds int

	flag.StringVar(&inputPath, "input", "", "Path to input CSV (defaults to stdin)")
	flag.IntVar(&numWorkers, "workers", 0, "Number of workers to use (defaults to number of cores)")
	flag.IntVar(&timeoutSeconds, "timeout", 0, "Timeout in seconds (defaults to no timeout)")
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

	if timeoutSeconds < 0 {
		flag.Usage()
		log.Fatalf("timeout must be greater than 0")
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	wp, err := workerpool.New(numWorkers)
	if err != nil {
		log.Fatalf("error creating worker pool: %v", err)
	}

	wp.RunWorkers(ctx)
	done := make(chan bool)
	go wp.SendMetrics(ctx, done)

	queryReader := query.NewReader(reader)
	for {
		select {
		case <-ctx.Done():
			log.Fatalf("Context cancelled: %v", ctx.Err())
		default:
		}

		query, hasMore, err := queryReader.Next()
		if err != nil {
			log.Fatalf("Error reading query: %v", err)
		}
		if !hasMore {
			break
		}

		if err := wp.RunQuery(ctx, query); err != nil {
			log.Fatalf("Error running query: %v", err)
		}
	}

	wp.Close()
	<-done

	metrics := wp.AggregateMetrics()
	fmt.Printf("metrics: %v\n", metrics.Table())
}
