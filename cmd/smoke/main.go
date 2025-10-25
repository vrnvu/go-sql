package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

// Smoke test the TigerData database connectivity
func main() {
	ctx := context.Background()
	connStr := "postgres://tigerdata:123@localhost:5432/tigerdata" //nolint:gosec
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	var greeting string
	err = conn.QueryRow(ctx, "select 'Hello, Timescale!'").Scan(&greeting)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(greeting)
}
