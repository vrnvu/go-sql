# Specs

## References:
I immidiatly thought about wrk, I've used it in the past. 
- [wrk](https://github.com/wg/wrk)
- [wrk2](https://github.com/giltene/wrk2)

## Functional
### Comptime
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
### Runtime
    - Mapping hostmap = worker (simple Round Robin baseline)
    - Error handling?
        - If something panics or context is cancel abort benchmark
        - What if a request to TigerData fails? Do we backoff/retry?   
    - Connections
        - Pool size, per-worker connections, simple backoff/retries?
    - Logging and aggregation
        - Structured JSON logs; per-request + aggregated
- Future enhancements
    - Take ideas from wrk/wrk2 (precise rate to TigerData)
    - Expand into e2e/integration tests
    - Expose as a lib/sdk (now we have a bin/cli, but it could be use to expose this as a library)

## Non Functional:
- Correctness: What happens if “somethings” panics?
    - If this is a benchmark tool, my first instinct is to consider the full test a failure. It needs to be fully repeated.
    - If otherwise we continued the test, i.e a worker panics but we spawn another to recover on runtime, the throughput and load in our target TigerData instance will be altered. Particularly affecting tail latency and throughput if system gets over-loaded.
    - Similarly, seems to me it doesn’t make sense to persist intermediate data to a file.
        - I thought about the resource limitation that our `.csv` file doesn't fit in memory. 
        - Also in other testing scenarios, testing correct logic, it would make sense to persist intermediate results in order to have snapshots and recoveries. This way we don’t need to re-trigger the full test if “something” panics.
- Observability / Metrics of the tool
    - Useful to have meta metrics about the tool itself, like retries to TigerData if some requests fail or even dropped.
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

## Design

High level:

<img width="834" height="406" alt="Screenshot 2025-10-23 at 09 52 59" src="https://github.com/user-attachments/assets/b83f7d9b-9455-4878-a8ca-f7bc8c597ce9" />

- cli: the cli sets the target, number of workers and input IO (stdin or file).
- csv reader / query generator: reads our input IO and generates queries, streams them to our worker pool as they are generated.
- worker pool: loadbalances to workers, the worker makes the http request and publishes results to a channel, aggregates metrics collected from the results


Worker details:

<img width="423" height="203" alt="Screenshot 2025-10-23 at 09 53 01" src="https://github.com/user-attachments/assets/8eee0b64-94ce-4c7c-b869-b6856c0c0cce" />

- Every worker will create its own connection to TigerData, so we will have N connections.
- Using bounded channels to simplify the CSP model, we can later profile and optimize with unbounded channels.
    - I was thinking about N worker : 1 results chan that blocks to make it simple and correct first. Profile and optimize in a second stage.

Connection:

<img width="376" height="474" alt="Screenshot 2025-10-23 at 09 53 03" src="https://github.com/user-attachments/assets/3cd3088c-8e9c-4e09-b235-64b72595b70e" />

- A worker will do a connection to TigerData to assert connectivity.
- Note: We could do here a first smoke test to get one row from the table to validate correctness of the system itself.
- After that we can start our benchmark, a simple design given 1 worker = 1 connection is to just send the queries sequentially and mesure timings.
- Collecting results is aggregating all our data t0, t1, ... tN, for every queryN.

------

Additional first thoughts, notes and details:

- IO heavy
    - Read from stdin or file
        - file: if fits memory we can mmap first then process
        - udated: according to the assignment the reading and parsing time of the queries is negligible, but in a real benchmark tool we care about total time of execution as well!
- Langs: Rust or Go
    - updated: I choose Go probably simpler if we focus only care on IO time of each request
- 1 core:thread per worker (baseline)
    - updated: With Go we can't distinguish M:N as we use `go run(...)` but will do if we spawn co-routines ourselves. 
- Simple Round Robin distribution between workers
    - Enhancements: other load balancing ideas
        - Push/Pull queues - workers strategy
- Map existing hosts to workers
    - updated; hostname => worker, we can use a round robin
- Pre-process query creation if everything fits MEM, simpler and more throughput
    - Load .csv, for every row convert to sql query, then run queries
    - We can store the sql queries in a snapshot if this is expensive
    - updated: negligible for assignment

Outputs
- Human summary table to stdout
- JSON file: metrics + config + env for reproducibility
