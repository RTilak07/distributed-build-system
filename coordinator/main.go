package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

	if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
		http.Error(w, "invalid worker registration", http.StatusBadRequest)
		return
	}

	if registration.ID == "" || registration.Address == "" {
		http.Error(w, "worker ID and address are required", http.StatusBadRequest)
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"worker":  registration.ID,
	})
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

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return shared.BuildResult{}, fmt.Errorf(
			"worker returned status %d",
			response.StatusCode,
		)
	}

	var result shared.BuildResult

	if err := json.Unmarshal(resultData, &result); err != nil {
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

	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "invalid build request", http.StatusBadRequest)
		return
	}

	if job.ID == "" {
		http.Error(w, "job ID is required", http.StatusBadRequest)
		return
	}

	if job.Project == "" {
		http.Error(w, "project is required", http.StatusBadRequest)
		return
	}

	fmt.Printf("Coordinator received job: %s\n", job.ID)

	cacheKey := generateCacheKey(job)

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
	var err error

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

		result, err = executeBuildOnWorker(worker, job)

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
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "coordinator is healthy")
}

func workersHandler(w http.ResponseWriter, r *http.Request) {
	count := buildScheduler.WorkerCount()
	healthyCount := buildScheduler.HealthyWorkerCount()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"worker_count":         count,
		"healthy_worker_count": healthyCount,
	})
}

func cacheHandler(w http.ResponseWriter, r *http.Request) {
	size := buildCache.Size()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"cache_entries": size,
	})
}

func runSafeDemoBuild() (shared.BuildResult, error) {
	projectPath := filepath.Join("sample-project")

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()

	result := shared.BuildResult{
		JobID:   fmt.Sprintf("demo-%d", time.Now().UnixNano()),
		Success: err == nil,
		Output:  string(output),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

func demoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	job := shared.BuildJob{
		ID:      fmt.Sprintf("demo-%d", time.Now().UnixNano()),
		Project: "sample-project",
		Status:  "queued",
	}

	fmt.Println()
	fmt.Println("========== PUBLIC DEMO BUILD ==========")
	fmt.Printf("Demo job: %s\n", job.ID)
	fmt.Printf("Project: %s\n", job.Project)

	cacheKey := generateCacheKey(job)

	if cachedResult, found := buildCache.Get(cacheKey); found {
		cachedResult.JobID = job.ID

		fmt.Println("Cache HIT")

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"result":       cachedResult,
			"cache":        "HIT",
			"worker":       "demo-worker-2",
			"attempts":     1,
			"demo_project": "sample-project",
			"mode":         "safe-public-demo",
		})

		return
	}

	fmt.Println("Cache MISS")
	fmt.Println("Demo mode: simulating worker-1 failure")
	fmt.Println("Attempt 1/3: worker-1 unavailable")
	fmt.Println("Worker worker-1 marked unhealthy")
	fmt.Println("Attempt 2/3: executing build on demo-worker-2")

	result, err := runSafeDemoBuild()

	if err != nil {
		fmt.Printf("Demo build failed: %v\n", err)

		http.Error(
			w,
			"safe demo build failed",
			http.StatusBadGateway,
		)

		return
	}

	result.JobID = job.ID

	buildCache.Set(cacheKey, result)

	fmt.Println("demo-worker-2 completed build successfully")
	fmt.Println("Result cached")
	fmt.Println("=======================================")

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"result":       result,
		"cache":        "MISS",
		"worker":       "demo-worker-2",
		"attempts":     2,
		"demo_project": "sample-project",
		"mode":         "safe-public-demo",
	})
}

func demoPageHandler(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("demo/index.html")

	if err != nil {
		http.Error(
			w,
			"demo interface unavailable",
			http.StatusInternalServerError,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	w.Write(data)
}

func main() {
	http.HandleFunc("/register", registerWorkerHandler)
	http.HandleFunc("/build", buildHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/workers", workersHandler)
	http.HandleFunc("/cache", cacheHandler)

	http.HandleFunc("/demo/build", demoHandler)
	http.HandleFunc("/demo", demoPageHandler)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	fmt.Printf(
		"Coordinator started on port %s\n",
		port,
	)

	fmt.Println("Demo available at /demo")

	err := http.ListenAndServe(
		":"+port,
		nil,
	)

	if err != nil {
		fmt.Println("Coordinator stopped:", err)
	}
}