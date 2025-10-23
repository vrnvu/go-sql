package metrics

import (
	"fmt"
	"slices"
	"time"
)

const (
	// ReservoirMaxCapacity is the maximum number of responses that can be stored in the Reservoir metrics
	ReservoirMaxCapacity = 20_000
	// ReservoirDefaultSampleSize is the fixed number of samples to keep in the reservoir
	ReservoirDefaultSampleSize = 10_000
)

// Reservoir is a sampling algorithm
type Reservoir struct {
	responses           []time.Duration
	numberOfQueries     int
	totalProcessingTime time.Duration
	minResponse         time.Duration
	maxResponse         time.Duration
	sampleSize          int
	funcRandIntn        func(n int) int
}

// NewReservoir creates a new Reservoir with default sample size
func NewReservoir(funcRandIntn func(n int) int) *Reservoir {
	return &Reservoir{
		responses:    make([]time.Duration, 0, ReservoirDefaultSampleSize),
		sampleSize:   ReservoirDefaultSampleSize,
		funcRandIntn: funcRandIntn,
	}
}

// NewReservoirWithSize creates a new Reservoir with specified sample size
func NewReservoirWithSize(sampleSize int, funcRandIntn func(n int) int) (*Reservoir, error) {
	if sampleSize < 1 {
		return nil, fmt.Errorf("sampleSize must be greater than 0")
	}

	if sampleSize > ReservoirMaxCapacity {
		return nil, fmt.Errorf("sampleSize must be less than %d", ReservoirMaxCapacity)
	}

	return &Reservoir{
		responses:    make([]time.Duration, 0, sampleSize),
		sampleSize:   sampleSize,
		funcRandIntn: funcRandIntn,
	}, nil
}

func (r *Reservoir) AddResponse(duration time.Duration) {
	r.numberOfQueries++
	r.totalProcessingTime += duration

	if duration < r.minResponse || r.numberOfQueries == 1 {
		r.minResponse = duration
	}
	if duration > r.maxResponse || r.numberOfQueries == 1 {
		r.maxResponse = duration
	}

	if len(r.responses) < r.sampleSize {
		r.responses = append(r.responses, duration)
	} else {
		// This is the reservoir sampling algorithm
		// And the main difference with a Simple metrics with fixed capacity
		// Simple panics.
		// Reservoir samples randomly and replaces the old value with the new one.
		j := r.funcRandIntn(r.numberOfQueries)
		if j < r.sampleSize {
			r.responses[j] = duration
		}
	}
}

// Aggregate aggregates the responses into a Result
func (r *Reservoir) Aggregate() Result {
	slices.Sort(r.responses)
	averageResponse := r.totalProcessingTime / time.Duration(r.numberOfQueries)

	var medianResponse time.Duration
	if len(r.responses) > 0 {
		medianIndex := len(r.responses) / 2
		medianResponse = r.responses[medianIndex]
	}

	return Result{
		NumberOfQueries:     r.numberOfQueries,
		TotalProcessingTime: r.totalProcessingTime,
		MinResponse:         r.minResponse,
		MedianResponse:      medianResponse,
		AverageResponse:     averageResponse,
		MaxResponse:         r.maxResponse,
	}
}
