package query

import (
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
	startTime, err := time.Parse(time.DateTime, "2025-01-01 00:00:00")
	assert.NoError(t, err)
	endTime, err := time.Parse(time.DateTime, "2025-01-01 00:00:01")
	assert.NoError(t, err)

	query := Query{
		Hostname:  "host1",
		StartTime: startTime,
		EndTime:   endTime,
	}

	snaps.MatchSnapshot(t, query.Build())
}
