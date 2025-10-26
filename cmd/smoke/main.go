package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vrnvu/go-sql/internal/client"
	"github.com/vrnvu/go-sql/internal/query"
)

// Smoke test the TigerData database connectivity
func main() {
	numberOfWorkers := 4
	ctx := context.Background()
	client, err := client.NewTigerData(ctx, numberOfWorkers, "tigerdata", "123", "localhost", "5432", "homework")
	if err != nil {
		log.Fatalf("Unable to create client: %v\n", err)
	}
	defer client.Close(ctx)

	// TODO: snapshot test
	query := query.Query{
		Hostname:  "host_000010",
		StartTime: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2017, 1, 1, 0, 1, 0, 0, time.UTC), // 1 minute
	}

	resp, err := client.Query(ctx, query.Build())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found: %v\n", resp.Duration)

	resp, err = client.Query(ctx, query.Build())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found: %v\n", resp.Duration)

	resp, err = client.Query(ctx, query.Build())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found: %v\n", resp.Duration)
}
