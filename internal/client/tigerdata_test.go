package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vrnvu/go-sql/internal/query"
)

func TestNewTigerPing(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: tigerdata ping")
	}
	t.Parallel()

	ctx := t.Context()
	numberOfWorkers := 2

	client, err := NewTigerData(ctx, numberOfWorkers, "tigerdata", "123", "localhost", "5432", "homework")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	defer client.Close(ctx)

	err = client.Ping(ctx)
	assert.NoError(t, err)
}

func TestTigerDataQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: tigerdata query")
	}
	t.Parallel()

	ctx := t.Context()
	numberOfWorkers := 2

	client, err := NewTigerData(ctx, numberOfWorkers, "tigerdata", "123", "localhost", "5432", "homework")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	defer client.Close(ctx)

	err = client.Ping(ctx)
	assert.NoError(t, err)

	query := query.Query{
		Hostname:  "host1",
		StartTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 1, 1, 0, 0, 1, 0, time.UTC),
	}

	resp, err := client.Query(ctx, query.Build())
	assert.NoError(t, err)
	assert.Greater(t, resp.Duration, 0*time.Second)
}
