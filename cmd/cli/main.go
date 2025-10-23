package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

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

	// TODO Since this is IO bound to Network, I'd consider a timeout
	ctx, cancel := context.WithCancel(context.Background())
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
			break // Finished reading
		}

		// TODO test context cancellation
		// if query.Hostname == "host_000002" {
		// 	cancel()
		// }

		if err := wp.RunQuery(ctx, query); err != nil {
			log.Fatalf("Error running query: %v", err)
		}
	}

	wp.Close()
	<-done

	metrics := wp.AggregateMetrics()
	fmt.Printf("metrics: %v\n", metrics)
}
