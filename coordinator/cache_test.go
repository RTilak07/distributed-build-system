package main

import (
	"fmt"
	"testing"

	"distributed-build-system/shared"
)

func BenchmarkBuildCacheSet(b *testing.B) {
	cache := NewBuildCache()

	result := shared.BuildResult{
		JobID:   "benchmark-job",
		Success: true,
		Output:  "benchmark build output",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("cache-key-%d", i)

		cache.Set(key, result)
	}
}

func BenchmarkBuildCacheGet(b *testing.B) {
	cache := NewBuildCache()

	result := shared.BuildResult{
		JobID:   "benchmark-job",
		Success: true,
		Output:  "benchmark build output",
	}

	cache.Set("benchmark-key", result)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Get("benchmark-key")
	}
}

func BenchmarkBuildCacheGetConcurrent(b *testing.B) {
	cache := NewBuildCache()

	result := shared.BuildResult{
		JobID:   "benchmark-job",
		Success: true,
		Output:  "benchmark build output",
	}

	cache.Set("benchmark-key", result)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get("benchmark-key")
		}
	})
}
