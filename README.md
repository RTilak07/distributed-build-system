\# Distributed Build System



A fault-tolerant distributed build execution system built with Go.



The system accepts build jobs through a central coordinator, schedules them across multiple workers, executes builds remotely, caches successful results, and automatically retries failed jobs using healthy workers.



\## Architecture



```text

&#x20;                        ┌──────────────────┐

&#x20;                        │     Developer    │

&#x20;                        │      / Client    │

&#x20;                        └────────┬─────────┘

&#x20;                                 │

&#x20;                                 │ HTTP

&#x20;                                 ▼

&#x20;                        ┌──────────────────┐

&#x20;                        │   Coordinator    │

&#x20;                        │                  │

&#x20;                        │  Job Management  │

&#x20;                        │  Cache           │

&#x20;                        │  Retry Logic     │

&#x20;                        └────────┬─────────┘

&#x20;                                 │

&#x20;                                 ▼

&#x20;                        ┌──────────────────┐

&#x20;                        │    Scheduler     │

&#x20;                        │                  │

&#x20;                        │ Round Robin      │

&#x20;                        │ Health Tracking  │

&#x20;                        │ Worker State     │

&#x20;                        └────────┬─────────┘

&#x20;                                 │

&#x20;                   ┌─────────────┼─────────────┐

&#x20;                   │             │             │

&#x20;                   ▼             ▼             ▼

&#x20;            ┌────────────┐ ┌────────────┐ ┌────────────┐

&#x20;            │  Worker 1  │ │  Worker 2  │ │  Worker 3  │

&#x20;            │            │ │            │ │            │

&#x20;            │ Go Build   │ │ Go Build   │ │ Go Build   │

&#x20;            └────────────┘ └────────────┘ └────────────┘



Key Features

\- Distributed build execution using multiple workers

\- Round-robin worker scheduling

\- Thread-safe in-memory build cache

\- SHA-256 based cache keys

\- Cache hits avoid repeated build execution

\- Worker health tracking

\- Automatic retry and worker failover

\- 30-second worker request timeout

\- Concurrent request handling

\- Worker registration through HTTP

\- Health monitoring endpoints

\- Automated unit tests

\- Go benchmarks for cache performance

How It Works

1\. Worker Registration

Workers register themselves with the coordinator when they start.

Worker → Coordinator /register



The coordinator stores the worker's ID and build endpoint.

2\. Build Request

A client sends a build job to:

POST /build



Example:

{

&#x20; "id": "build-001",

&#x20; "project": "./sample-project"

}



3\. Cache Lookup

Before assigning the job to a worker, the coordinator generates a SHA-256 cache key from the project input.

Build Request

&#x20;     │

&#x20;     ▼

&#x20;Cache Lookup

&#x20;  │       │

&#x20;HIT      MISS

&#x20; │         │

&#x20; ▼         ▼

Result    Scheduler

&#x20;           │

&#x20;           ▼

&#x20;         Worker



If a cached successful result exists, the coordinator returns it without executing another build.

4\. Worker Scheduling

The scheduler maintains worker state including:

\- Worker ID

\- Worker address

\- Busy state

\- Health state

Workers are selected using round-robin scheduling while skipping workers that are busy or unhealthy.

5\. Build Execution

The selected worker executes:

go test ./...



inside the requested project directory.

The worker returns:

\- Build success/failure

\- Build output

\- Error information

6\. Failure Handling

If a worker becomes unavailable:

Coordinator

&#x20;    │

&#x20;    ▼

&#x20;Worker 1

&#x20;    │

&#x20;    X  Request fails

&#x20;    │

&#x20;    ▼

Mark Worker 1 unhealthy

&#x20;    │

&#x20;    ▼

Retry

&#x20;    │

&#x20;    ▼

&#x20;Worker 2

&#x20;    │

&#x20;    ▼

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

│   ├── main\_test.go

│   ├── cache\_test.go

│   └── test\_helpers\_test.go

│

├── scheduler/

│   ├── scheduler.go

│   └── scheduler\_test.go

│

├── worker/

│   ├── main.go

│   └── main\_test.go

│

├── shared/

│   └── models.go

│

├── sample-project/

│   ├── go.mod

│   └── main.go

│

├── .gitignore

├── README.md

└── go.mod



Running the System

1\. Start the Coordinator

From the project root:

go run ./coordinator



The coordinator starts on:

http://localhost:8080



2\. Start Worker 1

Open another terminal:

go run ./worker -id worker-1 -port 8081



3\. Start Worker 2

Open another terminal:

go run ./worker -id worker-2 -port 8082



4\. Start Worker 3

Open another terminal:

go run ./worker -id worker-3 -port 8083



All workers automatically register with the coordinator.

Submitting a Build

From PowerShell:

Invoke-RestMethod `

&#x20; -Uri "http://localhost:8080/build" `

&#x20; -Method Post `

&#x20; -ContentType "application/json" `

&#x20; -Body '{"id":"build-001","project":"./sample-project"}'



A successful response looks like:

{

&#x20; "job\_id": "build-001",

&#x20; "success": true,

&#x20; "output": "..."

}



Checking Workers

Invoke-RestMethod "http://localhost:8080/workers"



Example:

{

&#x20; "worker\_count": 3,

&#x20; "healthy\_worker\_count": 3

}



Checking Cache

Invoke-RestMethod "http://localhost:8080/cache"



Example:

{

&#x20; "cache\_entries": 1

}



Testing

Run the complete test suite:

go test ./...



Current test coverage includes:

\- Worker registration

\- Invalid requests

\- Health endpoints

\- Worker scheduling

\- Busy worker handling

\- Unhealthy worker handling

\- Worker recovery

\- Cache behavior

\- Coordinator endpoints

\- Build execution

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

&#x20;  │

&#x20;  ▼

Cache Lookup

&#x20;  │

&#x20;  ├── HIT ───────────────► Return Cached Result

&#x20;  │

&#x20;  ▼

Scheduler

&#x20;  │

&#x20;  ▼

Healthy Worker

&#x20;  │

&#x20;  ├── Success ───────────► Cache Result

&#x20;  │

&#x20;  └── Failure

&#x20;         │

&#x20;         ▼

&#x20;   Mark Unhealthy

&#x20;         │

&#x20;         ▼

&#x20;      Retry

&#x20;         │

&#x20;         ▼

&#x20;  Another Healthy Worker



Future Improvements

Potential production-level improvements include:

\- Persistent distributed cache

\- Containerized build isolation

\- Persistent job queue

\- Worker heartbeat mechanism

\- Dynamic worker discovery

\- Authentication and authorization

\- Build artifact storage

\- Distributed coordinator instances

\- Metrics and observability

\- Job cancellation

\- Resource limits for build execution

Technology Stack

\- Go

\- HTTP

\- JSON

\- Goroutines / concurrent request handling

\- sync.RWMutex

\- SHA-256

\- Go testing framework

\- Go benchmarking framework

Author

R Tilak

