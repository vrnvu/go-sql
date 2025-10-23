package metrics

import "time"

// Result is the aggregated metrics for the Simple metrics
// - # of queries processed,
// - total processing time across all queries,
// - the minimum query time (for a single query),
// - the median query time,
// - the average query time,
// - and the maximum query time.
type Result struct {
	NumberOfQueries     int
	TotalProcessingTime time.Duration
	MinResponse         time.Duration
	MedianResponse      time.Duration
	AverageResponse     time.Duration
	MaxResponse         time.Duration
}
