package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os/exec"

	"distributed-build-system/shared"
)

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

	fmt.Printf("Received build job: %s\n", job.ID)
	fmt.Printf("Project: %s\n", job.Project)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = job.Project

	output, err := cmd.CombinedOutput()

	result := shared.BuildResult{
		JobID:   job.ID,
		Success: err == nil,
		Output:  string(output),
	}

	if err != nil {
		result.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	fmt.Printf(
		"Build completed: %s | Success: %v\n",
		job.ID,
		result.Success,
	)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "worker is healthy")
}

func registerWithCoordinator(
	workerID string,
	workerAddress string,
	coordinatorURL string,
) error {

	registration := shared.WorkerRegistration{
		ID:      workerID,
		Address: workerAddress + "/build",
	}

	data, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	response, err := http.Post(
		coordinatorURL+"/register",
		"application/json",
		bytes.NewBuffer(data),
	)

	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"registration failed with status %d",
			response.StatusCode,
		)
	}

	return nil
}

func main() {
	port := flag.String("port", "8081", "worker port")

	workerID := flag.String(
		"id",
		"worker-1",
		"worker ID",
	)

	coordinatorURL := flag.String(
		"coordinator",
		"http://localhost:8080",
		"coordinator URL",
	)

	flag.Parse()

	workerAddress := "http://localhost:" + *port

	http.HandleFunc("/build", buildHandler)
	http.HandleFunc("/health", healthHandler)

	err := registerWithCoordinator(
		*workerID,
		workerAddress,
		*coordinatorURL,
	)

	if err != nil {
		fmt.Println("Failed to register worker:", err)
		return
	}

	fmt.Printf(
		"Worker %s registered successfully\n",
		*workerID,
	)

	fmt.Printf(
		"Worker running on %s\n",
		workerAddress,
	)

	err = http.ListenAndServe(
		":"+*port,
		nil,
	)

	if err != nil {
		fmt.Println("Worker stopped:", err)
	}
}
