package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"distributed-build-system/scheduler"
	"distributed-build-system/shared"
)

var buildScheduler = scheduler.NewScheduler()
var buildCache = NewBuildCache()

const maxBuildAttempts = 3

func generateCacheKey(job shared.BuildJob) string {
	hash := sha256.Sum256([]byte(job.Project))

	return hex.EncodeToString(hash[:])
}

func registerWorkerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var registration shared.WorkerRegistration

	err := json.NewDecoder(r.Body).Decode(&registration)
	if err != nil {
		http.Error(w, "invalid worker registration", http.StatusBadRequest)
		return
	}

	if registration.ID == "" || registration.Address == "" {
		http.Error(
			w,
			"worker ID and address are required",
			http.StatusBadRequest,
		)
		return
	}

	buildScheduler.RegisterWorker(
		registration.ID,
		registration.Address,
	)

	fmt.Printf(
		"Worker registered: %s -> %s\n",
		registration.ID,
		registration.Address,
	)

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"success": true,
		"worker":  registration.ID,
	}

	json.NewEncoder(w).Encode(response)
}

func executeBuildOnWorker(
	worker *scheduler.Worker,
	job shared.BuildJob,
) (shared.BuildResult, error) {

	jobData, err := json.Marshal(job)

	if err != nil {
		return shared.BuildResult{}, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	response, err := client.Post(
		worker.Address,
		"application/json",
		bytes.NewBuffer(jobData),
	)

	if err != nil {
		return shared.BuildResult{}, err
	}

	defer response.Body.Close()

	resultData, err := io.ReadAll(response.Body)

	if err != nil {
		return shared.BuildResult{}, err
	}

	var result shared.BuildResult

	err = json.Unmarshal(resultData, &result)

	if err != nil {
		return shared.BuildResult{}, err
	}

	return result, nil
}

func buildHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var job shared.BuildJob

	err := json.NewDecoder(r.Body).Decode(&job)
	if err != nil {
		http.Error(w, "invalid build request", http.StatusBadRequest)
		return
	}

	fmt.Printf("Coordinator received job: %s\n", job.ID)

	cacheKey := generateCacheKey(job)

	fmt.Printf("Cache key: %s\n", cacheKey)

	cachedResult, found := buildCache.Get(cacheKey)

	if found {
		fmt.Printf("Cache HIT for job %s\n", job.ID)

		cachedResult.JobID = job.ID

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedResult)

		return
	}

	fmt.Printf("Cache MISS for job %s\n", job.ID)

	var result shared.BuildResult
	var lastError error

	for attempt := 1; attempt <= maxBuildAttempts; attempt++ {

		worker := buildScheduler.GetAvailableWorker()

		if worker == nil {
			lastError = fmt.Errorf("no healthy workers available")

			fmt.Printf(
				"Attempt %d/%d: no healthy workers available\n",
				attempt,
				maxBuildAttempts,
			)

			break
		}

		fmt.Printf(
			"Attempt %d/%d: assigning job %s to %s\n",
			attempt,
			maxBuildAttempts,
			job.ID,
			worker.ID,
		)

		result, err = executeBuildOnWorker(
			worker,
			job,
		)

		buildScheduler.ReleaseWorker(worker.ID)

		if err != nil {
			lastError = err

			buildScheduler.MarkWorkerUnhealthy(worker.ID)

			fmt.Printf(
				"Worker %s marked unhealthy: %v\n",
				worker.ID,
				err,
			)

			continue
		}

		fmt.Printf(
			"Worker %s completed job %s\n",
			worker.ID,
			job.ID,
		)

		lastError = nil
		break
	}

	if lastError != nil {
		http.Error(
			w,
			fmt.Sprintf(
				"build failed after %d attempts: %v",
				maxBuildAttempts,
				lastError,
			),
			http.StatusBadGateway,
		)

		return
	}

	if result.Success {
		buildCache.Set(cacheKey, result)

		fmt.Printf(
			"Cached successful result for job %s\n",
			job.ID,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	fmt.Printf(
		"Job %s completed successfully\n",
		job.ID,
	)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "coordinator is healthy")
}

func workersHandler(w http.ResponseWriter, r *http.Request) {
	count := buildScheduler.WorkerCount()
	healthyCount := buildScheduler.HealthyWorkerCount()

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"worker_count":         count,
		"healthy_worker_count": healthyCount,
	}

	json.NewEncoder(w).Encode(response)
}

func cacheHandler(w http.ResponseWriter, r *http.Request) {
	size := buildCache.Size()

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"cache_entries": size,
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/register", registerWorkerHandler)
	http.HandleFunc("/build", buildHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/workers", workersHandler)
	http.HandleFunc("/cache", cacheHandler)

	fmt.Println("Coordinator started on http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		fmt.Println("Coordinator stopped:", err)
	}
}
