# go-sql

Benchmarking tool designed to test TigerData (TimescaleDB) performance with concurrent queries. The tool reads query parameters from CSV files and executes them using a configurable worker pool.

## References TigerData:

- [Docker setup](https://docs.tigerdata.com/self-hosted/latest/install/installation-docker/)
- [Go client](https://docs.tigerdata.com/getting-started/latest/start-coding-with-timescale/)
- [Architecture Whitepaper](https://assets.timescale.com/docs/downloads/tigerdata-whitepaper.pdf)

## Quickstart

```bash
git clone https://github.com/vrnvu/go-sql
cd go-sql
make setup
make lint
make test
make docker-up
make test-slow
make run
make docker-down
```

## Usage

### CLI Tool
```bash
go run ./cmd/cli/main.go \
	-input ./resources/query_params.csv \
	-workers 64 \
	-timeout 10 \
	-db-user tigerdata \
	-db-password 123 \
	-db-host localhost \
	-db-port 5432 \
	-db-name homework
```

### Smoke Test

Ad-hoc client to local instace of Tigerdata.
```bash
go run ./cmd/smoke/main.go
```

## Testing
- `make test`: Run unit tests
- `make test-slow`: Run integration tests with property testing, DST (requires Docker)
- `make test-snap`: Update test snapshots
- `make test-cover`: Generate coverage report

# Specs

## Input Data Format

The tool expects a CSV file with the following format:
```csv
hostname,start_time,end_time
host_000008,2017-01-01 08:59:22,2017-01-01 09:59:22
host_000001,2017-01-02 13:02:02,2017-01-02 14:02:02
```

Each row represents a query to be executed against the TigerData database, filtering CPU usage data for a specific host within a time range.

Some basic data analytics on the distribution of the input.
```
1. host_000010: 17280 records
2. host_000017: 17280 records
3. host_000011: 17280 records
4. host_000000: 17280 records
5. host_000004: 17280 records
6. host_000007: 17280 records
7. host_000009: 17280 records
8. host_000014: 17280 records
9. host_000015: 17280 records
10. host_000016: 17280 records
```

- Since sample data is perfectly evenly distributed our workers should have exact loads, but it may be skewed in which our benchmark tool will have low throughput and under-used workers.

CPU:
```
CPU Usage Distribution:
Min: 0.00%
Max: 100.00%
Mean: 50.01%
Median: 49.99%
Standard Deviation: 833.48%
```
- This gives us an idea of expected results, in case we want to verify the queried results. I won't verify it.

Another observation from sample data is host appear in order:
```
2017-01-01 00:00:00,host_000000,92.77
2017-01-01 00:00:00,host_000001,76.26
2017-01-01 00:00:00,host_000002,6.94
2017-01-01 00:00:00,host_000003,76.03
...
2017-01-01 00:00:00,host_000017,28.05
2017-01-01 00:00:00,host_000018,12.09
2017-01-01 00:00:00,host_000019,90.35
```

- We are sequentially reading the csv and assigning to workers, and each worker does 1 request, and they appear in order
- Reading from file (DISK) should be faster than query (NET), so our csvreader shouldn't wait workers
- This affects how efficient the workerpool is.

## Architecture

The tool consists of several key components:

### Core Components
- CLI (`cmd/cli`): Main entry point with command-line argument parsing
- Smoke Test (`cmd/smoke`): Simple connectivity and query validation tool
- Client (`internal/client`): TigerData connection management with retry logic
- Worker Pool (`internal/workerpool`): Concurrent query execution with round-robin distribution
- Query Reader (`internal/query`): CSV parsing and query generation
- Metrics (`internal/metrics`): Performance measurement and aggregation. Two implementations.

### Performance Metrics Output
The tool provides detailed performance metrics in a formatted table:
```
=====================
Performance Metrics
=====================
Queries Processed: 200
Skipped Queries: 0
Failed Queries: 0
Total Time: 2.5s
Min Response: 1ms
Median Response: 5ms
Average Response: 12ms
Max Response: 45ms
```

## Functional Requirements

### Compile Time
- Read file: File exists and permissions to read
- Read stdin: Delimiter on new line '\n'
- Set TigerData target: Test connectivity first (smoke ping), test table schema is correct
- Set number of workers/clients: Hard limit workers (1024)

### Runtime
- Mapping hostmap = worker (simple Round Robin baseline)
- Error handling: If something panics or context is cancelled abort benchmark
- What if a request to TigerData fails: Retry instead of panic, then mark as failed
- Connections: N workers, 1 connection per worker, 3 retries without backoff
- Logging and aggregation: Simple logs, print data aggregation as table to stdout

## Design

### Features
- File input with CSV validation and error handling
- Stdin support for streaming input
- TigerData connection with ping validation
- Configurable worker pool (1-1024 workers)
- Round-robin query distribution with hostname mapping
- Retry logic for transient errors (3 attempts)
- Connection pooling (one connection per worker)
- Performance metrics aggregation
- Input validation with detailed error reporting
- Configurable query timeout
- Smoke test for connectivity validation

### Technical Details
- Client/pgx connection pool with configurable size
- Retry logic for connection issues and timeouts
- Hostname-to-worker mapping with round-robin fallback
- Error classification (skipped, failed, successful)

## Non-Functional Requirements

### Correctness & Reliability
- Graceful handling of individual query failures without stopping benchmark
- Automatic retry for transient database errors (connection issues, timeouts)
- Skip invalid CSV rows and continue processing
- Panic only on critical system failures affecting benchmark validity

### Performance & Scalability
- Support for 1-1024 concurrent workers
- Simple metrics and Reservoir implementations.
    - Simple stores in-memory (20,000 response capacity limit)
    - Reservoir uses a sampling technique to optimize efficiency
- Optimized for I/O-bound workloads

### Observability
- Track successful, failed, and skipped queries
- Min, max, median, and average response times
- Detailed error reporting and classification
- Real-time query processing status

### Security
- Basic auth psql 
- CSV format schema validation 
- Bounded memory usage to prevent resource exhaustion

### Testing and correctness
- Property testing
- Snapshot testing
- DST

## References:
- [wrk](https://github.com/wg/wrk)
- [wrk2](https://github.com/giltene/wrk2)