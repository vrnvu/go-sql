# Specs

## References:
I immediately thought about wrk, I've used it in the past. 
- [wrk](https://github.com/wg/wrk)
- [wrk2](https://github.com/giltene/wrk2)

## Sample data distribution:

I did some basic analysis to understand sample data and understand how the workers behavior is going to change depending on the input.  This is useful to generate multiple input samples that make sense statistically.

```
sources/cpu_usage.csv
Loaded 345600 records from resources/cpu_usage.csv

Host Distribution Analysis:
Total unique hosts: 20

Top 10 hosts by record count:
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

## References TigerData:

- [Docker setup](https://docs.tigerdata.com/self-hosted/latest/install/installation-docker/)
- [Go client](https://docs.tigerdata.com/getting-started/latest/start-coding-with-timescale/)
- [Architecture Whitepaper](https://assets.timescale.com/docs/downloads/tigerdata-whitepaper.pdf)

## Functional
### Comptime
    - Read file
        - Yes: File exists and permissions to read
        - No: Enforce optional limits (file size, max line length '\n')
    - Read stdin
        - Yes: Delimiter on new line '"\n"'
        - No: Max line length to prevent abuse
    - Set TigerData target
        - Yes: Test connectivity first (smoke ping)
        - Yes: Test table schema is correct
            - Smoke test: one row via prepared statement and validate serialization
    - Set number of workers/clients
        - Yes: Hard limit workers? 
            - Hard limit is 1024
### Runtime
    - Yes: Mapping hostmap = worker (simple Round Robin baseline)
    - Error handling?
        - Yes: If something panics or context is cancelled abort benchmark
        - Yes: What if a request to TigerData fails? Do we backoff/retry?   
            - It's important that we don't panic and retry instead, then mark as failed
    - Connections
        - Pool size, per-worker connections, simple backoff/retries?
            Yes: N workers, 1 connection per worker, 3 retries without backoff are fine
    - Logging and aggregation
        - Yes: Logs will be simple
        - No: We don't need fancy visualizations of data
        - Yes: Print data aggregation as a simple table string() to stdout
- Future enhancements (not needed now)
    - Take ideas from wrk/wrk2 (precise rate to TigerData)
    - Expand into e2e/integration tests
    - Expose as a lib/sdk (now we have a bin/cli, but it could be use to expose this as a library)
        - Note: Workerpool module needs re-work if we would like to expose as library so coroutine and ctx management is simpler

## Non Functional:
- Correctness: What happens if "something" panics?
    - Note: If this is a benchmark tool, my first instinct is to consider the full test a failure. It needs to be fully repeated.
    - Note: If otherwise we continued the test, i.e a worker panics but we spawn another to recover on runtime, the throughput and load in our target TigerData instance will be altered. Particularly affecting tail latency and throughput if system gets overloaded.
    - Yes: We will panic only when major events that affect the correctness of the system, i.e a worker completely crashes
    - Yes: Simple retries on HTTP queries to TigerData, skip invalid rows in .csv
    - No: No need to have snapshots or recoveries
    - Note: Similarly, seems to me it doesn’t make sense to persist intermediate data to a file.
        - I thought about the resource limitation that our `.csv` file doesn't fit in memory. 
        - Also in other testing scenarios, testing correct logic, it would make sense to persist intermediate results in order to have snapshots and recoveries. This way we don’t need to re-trigger the full test if “something” panics.
- Observability / Metrics of the tool
    - Useful to have meta metrics about the tool itself, like retries to TigerData if some requests fail or even dropped.
    - Yes: we want to have skipped csv rows, failed/retries of queries
- Resources to consider of the benchmarking instance:
    - CPU: Core / threads number, best ratio for M:N threading
        - Yes: For CPU bound One core - One worker, in this example IO bound probably we can go up to ~1024.
    - MEM:
        - Yes: Can we load the input .csv in memory?
        - Yes: Can we store and aggregate results in-memory?
        - Note: No hard-limit has been imposed. Still I think is good practice to have some upper bounds to avoid users allocating a slice of 10PB of RAM which will crash.
    - DISK: Any limits?
        - If MEM constraints we will need to consider efficient disk usage
        - No: We can assume reading from disk (.csv file) doesn't need particular attention
        - Yes: Only if reading a .csv row fails we can decide: crash or skip row
    - NETWORK: Connectivity, speed and bandwidth
        - No: no need for particular attention
- Security
    - What assumptions about tool security usages can we make?
    - Do we need to protect the TigerData instance from possible exploits in our CLI?
    - No: We do not need for particular attention

## Design (TODO)

High level:

<img width="834" height="406" alt="Screenshot 2025-10-23 at 09 52 59" src="https://github.com/user-attachments/assets/b83f7d9b-9455-4878-a8ca-f7bc8c597ce9" />

- cli: the cli sets the target, number of workers, timeout and input IO (stdin or file).
- csv reader / query generator: reads our input IO and generates queries, streams them to our worker pool as they are generated.
- worker pool: load balances to workers, the worker makes the http request and publishes results to a channel, aggregates metrics collected from the results


Worker details:

<img width="423" height="203" alt="Screenshot 2025-10-23 at 09 53 01" src="https://github.com/user-attachments/assets/8eee0b64-94ce-4c7c-b869-b6856c0c0cce" />

- Every worker will create its own connection to TigerData, so we will have N connections.
- Using bounded channels to simplify the CSP model, we can later profile and optimize with unbounded channels.
    - I was thinking about N worker : 1 results chan that blocks to make it simple and correct first. Profile and optimize in a second stage.
- Context timeout, I think makes sense. Consider failed request after Timeout, cancel HTTP request, process next query. 

Connection:

<img width="376" height="474" alt="Screenshot 2025-10-23 at 09 53 03" src="https://github.com/user-attachments/assets/3cd3088c-8e9c-4e09-b235-64b72595b70e" />

- A worker will do a connection to TigerData to assert connectivity.
- Note: We could do here a first smoke test to get one row from the table to validate correctness of the system itself.
- After that we can start our benchmark, a simple design given 1 worker = 1 connection is to just send the queries sequentially and measure timings.
- Collecting results is aggregating all our data t0, t1, ... tN, for every queryN.

## Metrics aggregation design


The module/functionality for metrics aggregation.

There are multiple possible implementations depending on our resource limits.

- Keeping a list of durations in a buffer of memory and compute everything at the end (doesn't scale).
- Sampling, we need to define an upper bound of max samples.
- Histograms, like wrk, we define buckets and assume a margin of error in median.

Min, max:
- Both of them can be simply tracked with a variable

For avg and median:
- Depends on metrics design
- avg = (avg * count * newDuration) / (count + 1)
    - Confirm if this math *always* works 
- median, depends on our aggregation approach and what % of error is acceptable
    - T-digest algorithm
    - P^2 algorithm
 
What we can notice this can be fully implemented and tested as a separate module and injected to our worker pool later.

------

Additional first thoughts, notes and details:

- IO heavy
    - Read from stdin or file
        - file: if fits memory we can mmap first then process
        - updated: according to the assignment the reading and parsing time of the queries is negligible, but in a real benchmark tool we care about total time of execution as well!
- Langs: Rust or Go
    - updated: I choose Go probably simpler if we focus only on IO time of each request
- 1 core:thread per worker (baseline)
    - updated: With Go we can't distinguish M:N as we use `go run(...)` but will do if we spawn co-routines ourselves. 
- Simple Round Robin distribution between workers
    - Enhancements: other load balancing ideas
        - Push/Pull queues - workers strategy
- Map existing hosts to workers
    - updated: hostname => worker, we can use a round robin
- Pre-process query creation if everything fits MEM, simpler and more throughput
    - Load .csv, for every row convert to sql query, then run queries
    - We can store the sql queries in a snapshot if this is expensive
    - updated: negligible for assignment

Outputs
- Human summary table to stdout
- JSON file: metrics + config + env for reproducibility
