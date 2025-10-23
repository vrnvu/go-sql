package client

import "time"

type Response struct {
	Duration time.Duration
}

type Client interface {
	Ping() error
	Query(query string) (*Response, error)
}
