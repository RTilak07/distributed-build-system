package shared

type BuildJob struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	Status  string `json:"status"`
}

type BuildResult struct {
	JobID   string `json:"job_id"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

type WorkerRegistration struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type CacheEntry struct {
	Key    string      `json:"key"`
	Result BuildResult `json:"result"`
}
