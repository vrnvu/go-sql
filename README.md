Specs

References:
- [wrk](https://github.com/wg/wrk)
- [wrk2](https://github.com/giltene/wrk2)

Functional:

- Comptime
    - Read file
        - File exists and permissions to read
        - Enforce optional limits (file size, max line length '\n')
    - Read stdin
        - Delimiter on new line '"\n"'
        - Max line length to prevent abuse
    - Set TigerData target
        - Test connectivity first (smoke ping)
        - Test table schema is correct
            - Smoke test: one row via prepared statement and validate serialization
    - Set number of workers/clients
        - Hard limit workers? 
            - i.e workers <= number_of_phisical_cores?
- Runtime
    - Mapping hostmap = worker (simple Round Robin baseline)
    - Error handling
    - For every row build SQL query
        - If fits memory, pre-process and batch for throughput
    - Rate control
        - Fixed-rate (wrk2-style token bucket) total
        - Cap max in-flight to avoid overload
    - Connections
        - Pool size, per-worker connections, simple backoff/retries
        - Idempotency strategy if retries (unique keys or staging table)
    - Logging and aggregation
        - Structured JSON logs; per-request + aggregated

- Enhancements
    - Take ideas from wrk/wrk2 (precise rate, staged loads)
    - Expand into e2e/integration tests
    - Expose as a lib/sdk

Non Functional:
- Correctness: What happens if “somethings” panics?
    - If this is a benchmark tool, my first instinct is to consider the full test a failure. It needs to be fully repeated.
    - If otherwise we continued the test, i.e a worker panics but we spawn another to recover on runtime, the throughput and load in our target TigerData instance will be altered. Particularly affecting tail latency and throughput if system gets over-loaded.
    - Similarly, seems to me it doesn’t make sense to persist intermediate data to a file.
        - In other testing scenarios, testing correct logic, it would make sense to persist intermediate results in order to have snapshots and recoveries. This way we don’t need to re-trigger the full test if “something” panics.
- Observability / Metrics
    - Latency p50/p95/p99, throughput (RPS), error rate
    - Optional HDR histogram
    - Record runner env (CPU cores, Go version), start/end time
- Resources to consider of the benchmarking instance:
    - CPU: Core / threads number, best ratio for M:N threading
    - MEM:
        - Can we load the input .csv in memory?
        - Can we store and aggregate results in-memory?
    - DISK: Any limits?
        - If MEM constraints we will need to consider efficient disk usage
    - NETWORK: Connectivity, speed and bandwidth
- Security
    - What assumptions about tool security usages can we make?
    - Do we need to protect the TigerData instance from possible exploits in our CLI?

Design

- IO heavy
    - Read from stdin or file
        - file: if fits memory we can mmap first then process
- Disk read → worker (do work) → DB write (NET IO)
- Langs: Rust or Go
    - Go probably simpler if we focus only care on IO time of each request
- 1 core:thread per worker (baseline)
- Simple Round Robin distribution between workers
    - Enhancements: other load balancing ideas
        - Push/Pull queues - workers strategy
- Map existing hosts to workers
    - String -> channel
    - Problem: skewed distribution
- Pre-process query creation if everything fits MEM, simpler and more throughput
    - Load .csv, for every row convert to sql query, then run queries
    - We can store the sql queries in a snapshot if this is expensive

Outputs

- Human summary table to stdout
- JSON file: metrics + config + env for reproducibility