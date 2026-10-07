package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"distributed-build-system/scheduler"
	"distributed-build-system/shared"
)

func resetCoordinatorState() {
	buildScheduler = scheduler.NewScheduler()
	buildCache = NewBuildCache()
}

func TestRegisterWorkerHandler(t *testing.T) {
	resetCoordinatorState()

	body := `{
		"id": "worker-test",
		"address": "http://localhost:9090/build"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	registerWorkerHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if buildScheduler.WorkerCount() != 1 {
		t.Fatalf(
			"expected 1 worker, got %d",
			buildScheduler.WorkerCount(),
		)
	}
}

func TestRegisterWorkerRejectsInvalidJSON(t *testing.T) {
	resetCoordinatorState()

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(`invalid-json`),
	)

	recorder := httptest.NewRecorder()

	registerWorkerHandler(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestRegisterWorkerRejectsMissingFields(t *testing.T) {
	resetCoordinatorState()

	body := `{
		"id": "worker-test"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	registerWorkerHandler(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	healthHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	expected := "coordinator is healthy\n"

	if recorder.Body.String() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			recorder.Body.String(),
		)
	}
}

func TestWorkersHandler(t *testing.T) {
	resetCoordinatorState()

	buildScheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	buildScheduler.RegisterWorker(
		"worker-2",
		"http://localhost:8082/build",
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workers",
		nil,
	)

	recorder := httptest.NewRecorder()

	workersHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response struct {
		WorkerCount        int `json:"worker_count"`
		HealthyWorkerCount int `json:"healthy_worker_count"`
	}

	err := json.NewDecoder(
		recorder.Body,
	).Decode(&response)

	if err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.WorkerCount != 2 {
		t.Fatalf(
			"expected 2 workers, got %d",
			response.WorkerCount,
		)
	}

	if response.HealthyWorkerCount != 2 {
		t.Fatalf(
			"expected 2 healthy workers, got %d",
			response.HealthyWorkerCount,
		)
	}
}

func TestCacheHandler(t *testing.T) {
	resetCoordinatorState()

	buildCache.Set(
		"test-cache-key",
		structBuildResult(),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/cache",
		nil,
	)

	recorder := httptest.NewRecorder()

	cacheHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response struct {
		CacheEntries int `json:"cache_entries"`
	}

	err := json.NewDecoder(
		recorder.Body,
	).Decode(&response)

	if err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.CacheEntries != 1 {
		t.Fatalf(
			"expected 1 cache entry, got %d",
			response.CacheEntries,
		)
	}
}

func structBuildResult() shared.BuildResult {
	return shared.BuildResult{
		JobID:   "test-job",
		Success: true,
		Output:  "test output",
	}
}
