package query

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

// Query is a row from the input CSV file, representing a single query to be executed
type Query struct {
	Hostname  string    `csv:"hostname"`
	StartTime time.Time `csv:"start_time"`
	EndTime   time.Time `csv:"end_time"`
}

// Reader is a simple iterator for reading CSV queries
type Reader struct {
	csvReader *csv.Reader
}

// NewReader creates a new query reader
func NewReader(csvReader *csv.Reader) *Reader {
	return &Reader{csvReader: csvReader}
}

// Next reads the next query from the CSV
// Returns the query and a boolean indicating if there are more queries
func (r *Reader) Next() (Query, bool, error) {
	record, err := r.csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return Query{}, false, nil
		}
		return Query{}, false, err
	}

	if len(record) != 3 {
		return Query{}, false, fmt.Errorf("invalid CSV record: expected 3 fields, got %d", len(record))
	}

	startTime, err := time.Parse("2006-01-02 15:04:05", record[1])
	if err != nil {
		return Query{}, false, fmt.Errorf("invalid start_time format: %v", err)
	}

	endTime, err := time.Parse("2006-01-02 15:04:05", record[2])
	if err != nil {
		return Query{}, false, fmt.Errorf("invalid end_time format: %v", err)
	}

	query := Query{
		Hostname:  record[0],
		StartTime: startTime,
		EndTime:   endTime,
	}

	return query, true, nil
}

// Build transforms the Query struct into the SQL query string
// We could build the query directly from the .csv file, but a Query struct give us flexibility to add more fields in the future and try different query patterns
// TODO
func (q *Query) Build() string {
	return fmt.Sprintf("SELECT * FROM cpu_usage WHERE hostname = %s AND ts BETWEEN %s AND %s", q.Hostname, q.StartTime, q.EndTime)
}
