package metrics

import (
	"fmt"
	"log"
	"slices"
	"time"
)

const (
	// SimpleMaxCapacity is the maximum number of responses that can be stored in the Simple metrics
	SimpleMaxCapacity = 1_000_000
)

// Simple metrics keeps everything in memory
// Not scalable, but simple and easy to implement and we can use it to verify correctness of other implemntations
type Simple struct {
	responses []time.Duration
}

// NewSimple creates a new Simple metrics
func NewSimple() *Simple {
	return &Simple{
		responses: make([]time.Duration, 0),
	}
}

// NewSimpleWithCapacity creates a new Simple metrics with a pre-allocated capacity
// In case we know the number of rows in the input CSV, we can pre-allocate the array to avoid re-allocations
// We have an upper bound of max samples, capacity as int could allocate PB of RAM
func NewSimpleWithCapacity(capacity int) (*Simple, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("capacity must be greater than 0")
	}

	if capacity > SimpleMaxCapacity {
		return nil, fmt.Errorf("capacity must be less than %d", SimpleMaxCapacity)
	}

	return &Simple{
		responses: make([]time.Duration, capacity),
	}, nil
}

// AddResponse adds a response duration to the Simple metrics
func (s *Simple) AddResponse(duration time.Duration) {
	if SimpleMaxCapacity == len(s.responses) {
		log.Panicf("Simple metrics has reached the max capacity of %d", SimpleMaxCapacity)
	}

	s.responses = append(s.responses, duration)
}

// Aggregate aggregates the responses into a Result
func (s *Simple) Aggregate() Result {
	slices.Sort(s.responses)

	numberOfQueries := len(s.responses)
	minResponse := s.responses[0]
	maxResponse := s.responses[0]
	totalProcessingTime := time.Duration(0)
	for _, response := range s.responses {
		if response < minResponse {
			minResponse = response
		}
		if response > maxResponse {
			maxResponse = response
		}
		totalProcessingTime += response
	}
	averageResponse := totalProcessingTime / time.Duration(numberOfQueries)
	medianResponse := s.responses[numberOfQueries/2]

	return Result{
		NumberOfQueries:     numberOfQueries,
		TotalProcessingTime: totalProcessingTime,
		MinResponse:         minResponse,
		MedianResponse:      medianResponse,
		AverageResponse:     averageResponse,
		MaxResponse:         maxResponse,
	}
}
