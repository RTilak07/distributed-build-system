package main

import (
	"distributed-build-system/scheduler"
)

func newTestScheduler() *scheduler.Scheduler {
	return scheduler.NewScheduler()
}

func newTestCache() *BuildCache {
	return NewBuildCache()
}
