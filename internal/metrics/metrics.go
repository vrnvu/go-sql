package metrics

import (
	"fmt"
	"strings"
	"time"
)

// Result is the aggregated metrics for the Simple metrics
// - # of queries processed,
// - total processing time across all queries,
// - the minimum query time (for a single query),
// - the median query time,
// - the average query time,
// - and the maximum query time.
type Result struct {
	NumberOfQueries     int
	SkippedQueries      int
	FailedQueries       int
	TotalProcessingTime time.Duration
	MinResponse         time.Duration
	MedianResponse      time.Duration
	AverageResponse     time.Duration
	MaxResponse         time.Duration
}

func (r *Result) Table() string {
	builder := strings.Builder{}
	builder.WriteString("\n\n=====================\n")
	builder.WriteString("Performance Metrics\n")
	builder.WriteString("=====================\n")
	builder.WriteString(fmt.Sprintf("Queries Processed: %d\n", r.NumberOfQueries))
	builder.WriteString(fmt.Sprintf("Skipped Queries: %d\n", r.SkippedQueries))
	builder.WriteString(fmt.Sprintf("Failed Queries: %d\n", r.FailedQueries))
	builder.WriteString(fmt.Sprintf("Total Time: %v\n", r.TotalProcessingTime))
	builder.WriteString(fmt.Sprintf("Min Response: %v\n", r.MinResponse))
	builder.WriteString(fmt.Sprintf("Median Response: %v\n", r.MedianResponse))
	builder.WriteString(fmt.Sprintf("Average Response: %v\n", r.AverageResponse))
	builder.WriteString(fmt.Sprintf("Max Response: %v\n", r.MaxResponse))
	return builder.String()
}
