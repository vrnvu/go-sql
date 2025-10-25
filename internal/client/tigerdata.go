package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// result is a single row from the query
type result struct {
	ts    time.Time
	host  string
	usage float64
}

func (r *result) String() string {
	return fmt.Sprintf("ts: %s, host: %s, usage: %f", r.ts.UTC().Format(time.DateTime), r.host, r.usage)
}

// TigerData client holds 1 connection to the database
type TigerData struct {
	conn *pgx.Conn
}

func NewTigerData(ctx context.Context, user, password, host, port, dbname string) (*TigerData, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname) //nolint:gosec
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	return &TigerData{conn: conn}, nil
}

func (t *TigerData) Close(ctx context.Context) error {
	return t.conn.Close(ctx)
}

// Ping tests the connection to the database and validates the schema
func (t *TigerData) Ping() error {
	return nil
}

func (t *TigerData) Query(ctx context.Context, query string) (*Response, error) {
	startTime := time.Now()
	rows, err := t.conn.Query(ctx, query)
	if err != nil {
		// TODO: handle retriable errors
		// TODO: handle non-retriable errors
		return nil, err
	}
	defer rows.Close()
	duration := time.Since(startTime)

	// TODO: sync.Pool optimization? slower worker means less throughput per worker
	var results []result
	for rows.Next() {
		var r result
		err := rows.Scan(&r.ts, &r.host, &r.usage)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	if rows.Err() != nil {
		// TODO: handle rows err if any
		return nil, rows.Err()
	}

	for _, r := range results {
		fmt.Println(r.String())
	}
	return &Response{Duration: duration}, nil
}
