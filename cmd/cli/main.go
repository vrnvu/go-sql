package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vrnvu/go-sql/internal/client"
	"github.com/vrnvu/go-sql/internal/query"
	"github.com/vrnvu/go-sql/internal/workerpool"
)

func main() {
	var inputPath string
	var numWorkers int
	var timeoutSeconds int

	flag.StringVar(&inputPath, "input", "", "Path to input CSV (defaults to stdin)")
	flag.IntVar(&numWorkers, "workers", 0, "Number of workers to use (defaults to number of cores)")
	flag.IntVar(&timeoutSeconds, "timeout", 600, "Timeout in seconds (defaults to 600 seconds)")
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

	if numWorkers < 1 || workerpool.MaxWorkers < numWorkers {
		flag.Usage()
		log.Fatalf("number of workers: %d must be greater than 0 and less than %d", numWorkers, workerpool.MaxWorkers)
	}

	if timeoutSeconds < 0 {
		flag.Usage()
		log.Fatalf("timeout must be greater than 0")
	}

	queryReader, err := query.NewQueryReader(reader)
	if err != nil {
		log.Fatalf("error reading query headers: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	client, err := client.NewTigerData(ctx, numWorkers, "tigerdata", "123", "localhost", "5432", "homework")
	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}
	defer client.Close(ctx)

	if err := client.Ping(ctx); err != nil {
		log.Fatalf("error pinging client: %v", err)
	}

	wp, err := workerpool.New(numWorkers, client, queryReader)
	if err != nil {
		log.Fatalf("error creating worker pool: %v", err)
	}

	metrics, err := wp.Run(ctx)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("%v\n", metrics.Table())
}
