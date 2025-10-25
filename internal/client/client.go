package client

import (
	"context"
	"time"
)

type Response struct {
	Duration time.Duration
}

type Client interface {
	Ping() error
	Query(ctx context.Context, query string) (*Response, error)
}
