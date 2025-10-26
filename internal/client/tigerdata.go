package client

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: unused, do we need to verify anything?
// result is a single row from the query
// type result struct {
// 	ts    time.Time
// 	host  string
// 	usage float64
// }

// func (r *result) String() string {
// 	return fmt.Sprintf("ts: %s, host: %s, usage: %f", r.ts.UTC().Format(time.DateTime), r.host, r.usage)
// }

// TigerData client holds a connection pool to the database
type TigerData struct {
	pool *pgxpool.Pool
}

func NewTigerData(ctx context.Context, numberOfWorkers int, user, password, host, port, dbname string) (*TigerData, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname) //nolint:gosec

	// Configure connection pool
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Set pool size to fixed number of workers
	config.MaxConns = int32(numberOfWorkers) //nolint:gosec
	config.MinConns = int32(numberOfWorkers) //nolint:gosec

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return &TigerData{pool: pool}, nil
}

// Close closes the connection pool
func (t *TigerData) Close() error {
	t.pool.Close()
	return nil
}

// Ping tests the connection to the database and validates the schema
func (t *TigerData) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return t.pool.Ping(ctx)
}

func (t *TigerData) Query(ctx context.Context, query string) (*Response, error) {
	startTime := time.Now()

	maxRetries := 3
	var lastErr error

	for attempt := range maxRetries {
		rows, err := t.pool.Query(ctx, query)
		if err != nil {
			lastErr = err
			if isRetriableError(err) && attempt < maxRetries-1 {
				log.Printf("tigerdata query error: %v, retrying...", err)
				continue
			}
			return nil, lastErr
		}

		if rows.Err() != nil {
			log.Printf("tigerdata query error: %v", rows.Err())
			rows.Close()
			return nil, rows.Err()
		}

		rows.Close()

		duration := time.Since(startTime)
		return &Response{Duration: duration}, nil
	}

	return nil, lastErr
}

func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// TODO: tigerdata documentation on retriable errors
	// https://www.tigerdata.com/blog/5-common-connection-errors-in-postgresql-and-how-to-solve-them
	retriableErrors := []string{
		"conn busy",
		"connection reset",
		"connection refused",
		"timeout",
		"temporary failure",
		"server closed the connection",
		"broken pipe",
	}

	for _, retriableErr := range retriableErrors {
		if strings.Contains(errStr, retriableErr) {
			return true
		}
	}

	return false
}
