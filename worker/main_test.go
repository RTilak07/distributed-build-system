package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	expected := "worker is healthy\n"

	if recorder.Body.String() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			recorder.Body.String(),
		)
	}
}

func TestBuildHandlerRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/build",
		strings.NewReader(`invalid-json`),
	)

	recorder := httptest.NewRecorder()

	buildHandler(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestBuildHandlerRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/build",
		nil,
	)

	recorder := httptest.NewRecorder()

	buildHandler(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}

func TestBuildHandlerBuildsSampleProject(t *testing.T) {
	body := `{
		"id": "worker-test-job",
		"project": "../sample-project"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/build",
		strings.NewReader(body),
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	buildHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response struct {
		JobID   string `json:"job_id"`
		Success bool   `json:"success"`
		Output  string `json:"output"`
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

	if response.JobID != "worker-test-job" {
		t.Fatalf(
			"expected job ID worker-test-job, got %s",
			response.JobID,
		)
	}

	if !response.Success {
		t.Fatalf(
			"expected build to succeed, output: %s",
			response.Output,
		)
	}
}
