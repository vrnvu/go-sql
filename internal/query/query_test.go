package query

import (
	"encoding/csv"
	"os"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
)

func TestQuerySnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: query snapshot")
	}

	t.Parallel()
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 1, 0, 0, 1, 0, time.UTC)

	query := Query{
		Hostname:  "host1",
		StartTime: startTime,
		EndTime:   endTime,
	}

	snaps.MatchSnapshot(t, query.Build())
}

func AssertHeaders(t *testing.T, fields []string) {
	assert.Equal(t, 3, len(fields))
	assert.Equal(t, "hostname", fields[0])
	assert.Equal(t, "start_time", fields[1])
	assert.Equal(t, "end_time", fields[2])
}

func TestFieldsSnapshot(t *testing.T) {
	t.Parallel()
	inputFile, err := os.Open("../../resources/query_params.csv")
	assert.NoError(t, err)
	defer inputFile.Close()

	reader := csv.NewReader(inputFile)
	fields, err := reader.Read()
	assert.NoError(t, err)
	AssertHeaders(t, fields)
}

func TestQueryInvalidHeaderSnapshot(t *testing.T) {
	t.Parallel()
	inputFile, err := os.Open("../../resources/invalid_headers.csv")
	assert.NoError(t, err)
	defer inputFile.Close()

	reader := csv.NewReader(inputFile)
	queryReader, err := NewQueryReader(reader)
	assert.Error(t, err)
	assert.Nil(t, queryReader)
	snaps.MatchSnapshot(t, err.Error())
}

func TestQueryInvalidRowSnapshot(t *testing.T) {
	t.Parallel()
	inputFile, err := os.Open("../../resources/invalid_row.csv")
	assert.NoError(t, err)
	defer inputFile.Close()

	// line 1: valid headers
	reader := csv.NewReader(inputFile)
	queryReader, err := NewQueryReader(reader)
	assert.NoError(t, err)
	assert.NotNil(t, queryReader)

	// line 2: valid
	query, hasMore, err := queryReader.Next()
	assert.NoError(t, err)
	assert.NotNil(t, query)
	assert.True(t, hasMore)
	snaps.MatchSnapshot(t, query.Build())

	// line 3: invalid value
	query, hasMore, err = queryReader.Next()
	assert.Error(t, err)
	assert.Empty(t, query)
	assert.True(t, hasMore)
	snaps.MatchSnapshot(t, err.Error())

	// line 4: extra field
	query, hasMore, err = queryReader.Next()
	assert.Error(t, err)
	assert.Empty(t, query)
	assert.False(t, hasMore)
	snaps.MatchSnapshot(t, err.Error())
}
