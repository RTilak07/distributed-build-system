package scheduler

import "testing"

func TestRegisterWorker(t *testing.T) {
	scheduler := NewScheduler()

	scheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	if scheduler.WorkerCount() != 1 {
		t.Fatalf(
			"expected 1 worker, got %d",
			scheduler.WorkerCount(),
		)
	}

	scheduler.RegisterWorker(
		"worker-2",
		"http://localhost:8082/build",
	)

	if scheduler.WorkerCount() != 2 {
		t.Fatalf(
			"expected 2 workers, got %d",
			scheduler.WorkerCount(),
		)
	}
}

func TestRoundRobinScheduling(t *testing.T) {
	scheduler := NewScheduler()

	scheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	scheduler.RegisterWorker(
		"worker-2",
		"http://localhost:8082/build",
	)

	scheduler.RegisterWorker(
		"worker-3",
		"http://localhost:8083/build",
	)

	worker1 := scheduler.GetAvailableWorker()

	if worker1 == nil {
		t.Fatal("expected worker-1, got nil")
	}

	if worker1.ID != "worker-1" {
		t.Fatalf(
			"expected worker-1, got %s",
			worker1.ID,
		)
	}

	scheduler.ReleaseWorker(worker1.ID)

	worker2 := scheduler.GetAvailableWorker()

	if worker2 == nil {
		t.Fatal("expected worker-2, got nil")
	}

	if worker2.ID != "worker-2" {
		t.Fatalf(
			"expected worker-2, got %s",
			worker2.ID,
		)
	}

	scheduler.ReleaseWorker(worker2.ID)

	worker3 := scheduler.GetAvailableWorker()

	if worker3 == nil {
		t.Fatal("expected worker-3, got nil")
	}

	if worker3.ID != "worker-3" {
		t.Fatalf(
			"expected worker-3, got %s",
			worker3.ID,
		)
	}
}

func TestBusyWorkerIsSkipped(t *testing.T) {
	scheduler := NewScheduler()

	scheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	scheduler.RegisterWorker(
		"worker-2",
		"http://localhost:8082/build",
	)

	worker1 := scheduler.GetAvailableWorker()

	if worker1 == nil {
		t.Fatal("expected worker-1, got nil")
	}

	worker2 := scheduler.GetAvailableWorker()

	if worker2 == nil {
		t.Fatal("expected worker-2, got nil")
	}

	if worker1.ID == worker2.ID {
		t.Fatal("scheduler assigned the same busy worker twice")
	}
}

func TestUnhealthyWorkerIsSkipped(t *testing.T) {
	scheduler := NewScheduler()

	scheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	scheduler.RegisterWorker(
		"worker-2",
		"http://localhost:8082/build",
	)

	scheduler.MarkWorkerUnhealthy("worker-1")

	worker := scheduler.GetAvailableWorker()

	if worker == nil {
		t.Fatal("expected a healthy worker, got nil")
	}

	if worker.ID != "worker-2" {
		t.Fatalf(
			"expected worker-2, got %s",
			worker.ID,
		)
	}
}

func TestWorkerRecovery(t *testing.T) {
	scheduler := NewScheduler()

	scheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	scheduler.MarkWorkerUnhealthy("worker-1")

	scheduler.RegisterWorker(
		"worker-1",
		"http://localhost:8081/build",
	)

	worker := scheduler.GetAvailableWorker()

	if worker == nil {
		t.Fatal("expected recovered worker, got nil")
	}

	if worker.ID != "worker-1" {
		t.Fatalf(
			"expected worker-1, got %s",
			worker.ID,
		)
	}
}
