package query

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"time"
)

// Reader is a simple iterator for reading queries
type Reader interface {
	Next() (Query, bool, error)
}

// Query is a row from the input CSV file, representing a single query to be executed
type Query struct {
	Hostname  string    `csv:"hostname"`
	StartTime time.Time `csv:"start_time"`
	EndTime   time.Time `csv:"end_time"`
}

// CSVReader is a simple iterator for reading CSV queries
type CSVReader struct {
	csvReader *csv.Reader
	line      int
}

// NewReader creates a new query reader and validates the headers
func NewQueryReader(csvReader *csv.Reader) (*CSVReader, error) {
	fields, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading headers: %w", err)
	}
	if len(fields) != 3 {
		return nil, fmt.Errorf("expected 3 fields, got %d", len(fields))
	}
	if fields[0] != "hostname" || fields[1] != "start_time" || fields[2] != "end_time" {
		return nil, fmt.Errorf("expected fields to be hostname, start_time, end_time, got %v", fields)
	}

	return &CSVReader{csvReader: csvReader, line: 2}, nil
}

// Next reads the next query from the CSV
// Returns the query and a boolean indicating if there are more queries
// Skips errors when reading invalid rows
func (r *CSVReader) Next() (Query, bool, error) {
	defer func() {
		r.line++
	}()

	record, err := r.csvReader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return Query{}, false, nil
		}
		return Query{}, false, fmt.Errorf("error reading CSV record: %w on line %d", err, r.line)
	}

	if len(record) != 3 {
		return Query{}, true, fmt.Errorf("invalid CSV record: expected 3 fields, got %d on line %d", len(record), r.line)
	}

	// TODO: we are not validating hostname format

	startTime, err := time.Parse("2006-01-02 15:04:05", record[1])
	if err != nil {
		return Query{}, true, fmt.Errorf("invalid start_time: %s err: %w on line %d", record[1], err, r.line)
	}

	endTime, err := time.Parse("2006-01-02 15:04:05", record[2])
	if err != nil {
		return Query{}, true, fmt.Errorf("invalid end_time: %s err: %w on line %d", record[2], err, r.line)
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
