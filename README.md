# Distributed Build System

A fault-tolerant distributed build execution system built with Go.

The system accepts build jobs through a central coordinator, schedules them across multiple workers, executes builds remotely, caches successful results, and automatically retries failed jobs using healthy workers.

## Architecture

![Distributed Build System Architecture](docs/architecture.png)

```text
                         ┌──────────────────┐
                         │     Developer    │
                         │      / Client    │
                         └────────┬─────────┘
                                  │
                                  │ HTTP
                                  ▼
                         ┌──────────────────┐
                         │   Coordinator    │
                         │                  │
                         │  Job Management  │
                         │  Cache           │
                         │  Retry Logic     │
                         └────────┬─────────┘
                                  │
                                  ▼
                         ┌──────────────────┐
                         │    Scheduler     │
                         │                  │
                         │ Round Robin      │
                         │ Health Tracking  │
                         │ Worker State     │
                         └────────┬─────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
                    ▼             ▼             ▼
             ┌────────────┐ ┌────────────┐ ┌────────────┐
             │  Worker 1  │ │  Worker 2  │ │  Worker 3  │
             │            │ │            │ │            │
             │ Go Build   │ │ Go Build   │ │ Go Build   │
             └────────────┘ └────────────┘ └────────────┘
## Demo

### Worker Health

Three workers registered successfully with the coordinator.

![Worker Health](docs/screenshots/worker-health.png)

### Scheduling and Caching

The scheduler distributes builds across workers while cached projects are served without another worker execution.

![Scheduling and Cache](docs/screenshots/scheduling-cache.png)

### Failure Recovery

When a worker becomes unavailable, the coordinator retries the build using another healthy worker.

![Worker Failover](docs/screenshots/failover.png)
Key Features
- Distributed build execution using multiple workers
- Round-robin worker scheduling
- Thread-safe in-memory build cache
- SHA-256 based cache keys
- Cache hits avoid repeated build execution
- Worker health tracking
- Automatic retry and worker failover
- 30-second worker request timeout
- Concurrent request handling
- Worker registration through HTTP
- Health monitoring endpoints
- Automated unit tests
- Go benchmarks for cache performance
How It Works
1. Worker Registration
Workers register themselves with the coordinator when they start.
Worker → Coordinator /register

The coordinator stores the worker's ID and build endpoint.
2. Build Request
A client sends a build job to:
POST /build

Example:
{
  "id": "build-001",
  "project": "./sample-project"
}

3. Cache Lookup
Before assigning the job to a worker, the coordinator generates a SHA-256 cache key from the project input.
Build Request
      │
      ▼
 Cache Lookup
   │       │
 HIT      MISS
  │         │
  ▼         ▼
Result    Scheduler
            │
            ▼
          Worker

If a cached successful result exists, the coordinator returns it without executing another build.
4. Worker Scheduling
The scheduler maintains worker state including:
- Worker ID
- Worker address
- Busy state
- Health state
Workers are selected using round-robin scheduling while skipping workers that are busy or unhealthy.
5. Build Execution
The selected worker executes:
go test ./...

inside the requested project directory.
The worker returns:
- Build success/failure
- Build output
- Error information
6. Failure Handling
If a worker becomes unavailable:
Coordinator
     │
     ▼
 Worker 1
     │
     X  Request fails
     │
     ▼
Mark Worker 1 unhealthy
     │
     ▼
Retry
     │
     ▼
 Worker 2
     │
     ▼
Successful Build

The coordinator supports up to 3 build attempts and avoids unhealthy workers during subsequent scheduling.
API Endpoints
Coordinator
Method	Endpoint	Description
POST	/register	Register a worker
POST	/build	Submit a build job
GET	/health	Coordinator health check
GET	/workers	Worker count and health status
GET	/cache	Cache entry count


Worker
Method	Endpoint	Description
POST	/build	Execute a build
GET	/health	Worker health check


Project Structure
distributed-build-system/
│
├── coordinator/
│   ├── main.go
│   ├── cache.go
│   ├── main_test.go
│   ├── cache_test.go
│   └── test_helpers_test.go
│
├── scheduler/
│   ├── scheduler.go
│   └── scheduler_test.go
│
├── worker/
│   ├── main.go
│   └── main_test.go
│
├── shared/
│   └── models.go
│
├── sample-project/
│   ├── go.mod
│   └── main.go
│
├── docs/
│   └── architecture.png
│
├── .gitignore
├── README.md
└── go.mod

Running the System
1. Start the Coordinator
From the project root:
go run ./coordinator

The coordinator starts on:
http://localhost:8080

2. Start Worker 1
Open another terminal:
go run ./worker -id worker-1 -port 8081

3. Start Worker 2
Open another terminal:
go run ./worker -id worker-2 -port 8082

4. Start Worker 3
Open another terminal:
go run ./worker -id worker-3 -port 8083

All workers automatically register with the coordinator.
Submitting a Build
From PowerShell:
Invoke-RestMethod `
  -Uri "http://localhost:8080/build" `
  -Method Post `
  -ContentType "application/json" `
  -Body '{"id":"build-001","project":"./sample-project"}'

A successful response looks like:
{
  "job_id": "build-001",
  "success": true,
  "output": "..."
}

Checking Workers
Invoke-RestMethod "http://localhost:8080/workers"

Example:
{
  "worker_count": 3,
  "healthy_worker_count": 3
}

Checking Cache
Invoke-RestMethod "http://localhost:8080/cache"

Example:
{
  "cache_entries": 1
}

Testing
Run the complete test suite:
go test ./...

Current test coverage includes:
- Worker registration
- Invalid requests
- Health endpoints
- Worker scheduling
- Busy worker handling
- Unhealthy worker handling
- Worker recovery
- Cache behavior
- Coordinator endpoints
- Build execution
Benchmark
The build cache was benchmarked using Go's built-in benchmarking framework.
Test environment:
OS:      Windows
Arch:    amd64
CPU:     11th Gen Intel Core i5-11260H @ 2.60GHz

Results:
BenchmarkBuildCacheSet-12
2,619,244 iterations
471.6 ns/op
285 B/op
2 allocs/op

BenchmarkBuildCacheGet-12
49,291,029 iterations
23.71 ns/op
0 B/op
0 allocs/op

BenchmarkBuildCacheGetConcurrent-12
25,920,388 iterations
42.83 ns/op
0 B/op
0 allocs/op

The concurrent benchmark uses Go's parallel benchmark execution to measure cache reads under concurrent access.
Design Decisions
In-Memory Cache
The current cache uses a thread-safe Go map protected by sync.RWMutex.
This keeps the system lightweight while demonstrating concurrent access control.
A production deployment could replace this with a distributed cache such as Redis when cache state needs to survive coordinator restarts or be shared between coordinator instances.
Round-Robin Scheduling
Round-robin scheduling provides simple and predictable distribution of work across workers.
The scheduler also tracks worker availability and health so failed workers are skipped.
Worker Failover
Workers that fail during build execution are marked unhealthy.
Subsequent retry attempts select another healthy worker when available.
HTTP Communication
The coordinator and workers communicate using HTTP and JSON.
This keeps the components independently deployable and makes the system easy to inspect and test.
Reliability Model
The coordinator provides several reliability mechanisms:
Request
   │
   ▼
Cache Lookup
   │
   ├── HIT ───────────────► Return Cached Result
   │
   ▼
Scheduler
   │
   ▼
Healthy Worker
   │
   ├── Success ───────────► Cache Result
   │
   └── Failure
          │
          ▼
    Mark Unhealthy
          │
          ▼
       Retry
          │
          ▼
   Another Healthy Worker

Future Improvements
Potential production-level improvements include:
- Persistent distributed cache
- Containerized build isolation
- Persistent job queue
- Worker heartbeat mechanism
- Dynamic worker discovery
- Authentication and authorization
- Build artifact storage
- Distributed coordinator instances
- Metrics and observability
- Job cancellation
- Resource limits for build execution
Technology Stack
- Go
- HTTP
- JSON
- Goroutines / concurrent request handling
- sync.RWMutex
- SHA-256
- Go testing framework
- Go benchmarking framework
Author
R Tilak