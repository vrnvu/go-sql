package query

import (
	"fmt"
	"time"
)

// Query is a row from the input CSV file, representing a single query to be executed
type Query struct {
	Hostname  string    `csv:"hostname"`
	StartTime time.Time `csv:"start_time"`
	EndTime   time.Time `csv:"end_time"`
}

// Build transforms the Query struct into the SQL query string
// We could build the query directly from the .csv file, but a Query struct give us flexibility to add more fields in the future and try different query patterns
// TODO
func (q *Query) Build() string {
	return fmt.Sprintf("SELECT * FROM cpu_usage WHERE hostname = %s AND ts BETWEEN %s AND %s", q.Hostname, q.StartTime, q.EndTime)
}
