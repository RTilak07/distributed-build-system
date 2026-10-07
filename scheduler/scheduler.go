package scheduler

import "sync"

type Worker struct {
	ID      string
	Address string
	Busy    bool
	Healthy bool
}

type Scheduler struct {
	workers    []Worker
	nextWorker int
	mu         sync.Mutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		workers: make([]Worker, 0),
	}
}

func (s *Scheduler) RegisterWorker(workerID string, address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.workers {
		if s.workers[i].ID == workerID {
			s.workers[i].Address = address
			s.workers[i].Busy = false
			s.workers[i].Healthy = true
			return
		}
	}

	s.workers = append(s.workers, Worker{
		ID:      workerID,
		Address: address,
		Busy:    false,
		Healthy: true,
	})
}

func (s *Scheduler) GetAvailableWorker() *Worker {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.workers) == 0 {
		return nil
	}

	for i := 0; i < len(s.workers); i++ {
		index := (s.nextWorker + i) % len(s.workers)

		if !s.workers[index].Busy &&
			s.workers[index].Healthy {

			s.workers[index].Busy = true

			s.nextWorker = (index + 1) % len(s.workers)

			worker := s.workers[index]

			return &worker
		}
	}

	return nil
}

func (s *Scheduler) ReleaseWorker(workerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.workers {
		if s.workers[i].ID == workerID {
			s.workers[i].Busy = false
			return
		}
	}
}

func (s *Scheduler) MarkWorkerUnhealthy(workerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.workers {
		if s.workers[i].ID == workerID {
			s.workers[i].Busy = false
			s.workers[i].Healthy = false
			return
		}
	}
}

func (s *Scheduler) WorkerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.workers)
}

func (s *Scheduler) HealthyWorkerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0

	for _, worker := range s.workers {
		if worker.Healthy {
			count++
		}
	}

	return count
}
