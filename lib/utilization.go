package lib

import (
	"math"
	"sync/atomic"
)

// WorkerUtilization tracks per-worker "hits" (invocations) and reports what percentage
// of workers were active over a sliding window of ticks. Workers call Poke(workerIdx)
// when they do work; the progress loop calls Tick() each interval to get the utilized percent.
type WorkerUtilization struct {
	hits        []int32
	history     [][]int32
	windowTicks int
}

// NewWorkerUtilization creates a tracker for numWorkers. windowTicks is how many
// Tick() calls to keep; utilization is the percent of workers with at least one hit
// since the oldest tick in the window (e.g. 10 ticks ≈ 1 second at 100ms).
func NewWorkerUtilization(numWorkers, windowTicks int) *WorkerUtilization {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if windowTicks <= 0 {
		windowTicks = 10
	}
	return &WorkerUtilization{
		hits:        make([]int32, numWorkers),
		history:     nil,
		windowTicks: windowTicks,
	}
}

// Poke records that the given worker did a unit of work. Safe to call from any goroutine.
// If workerIdx is out of range, Poke is a no-op.
func (u *WorkerUtilization) Poke(workerIdx int) {
	if workerIdx < 0 || workerIdx >= len(u.hits) {
		return
	}
	atomic.AddInt32(&u.hits[workerIdx], 1)
}

// Tick takes a snapshot of current hits, adds it to the window, and returns the
// percentage of workers that had at least one hit since the oldest snapshot in the
// window. Return value is rounded up to a whole percent (0–100). Call from a single
// goroutine (e.g. the progress loop).
func (u *WorkerUtilization) Tick() int {
	n := len(u.hits)
	if n == 0 {
		return 0
	}
	cur := make([]int32, n)
	for i := range u.hits {
		cur[i] = atomic.LoadInt32(&u.hits[i])
	}
	u.history = append(u.history, cur)
	if len(u.history) > u.windowTicks {
		u.history = u.history[1:]
	}
	active := 0
	if len(u.history) >= 2 {
		oldest := u.history[0]
		for i := range cur {
			if cur[i] > oldest[i] {
				active++
			}
		}
	} else {
		for _, c := range cur {
			if c > 0 {
				active++
			}
		}
	}
	return int(math.Ceil(100.0 * float64(active) / float64(n)))
}

// UtilizedPercentWholeRun returns the percentage of workers that had at least one
// Poke over the whole run so far (rounded up to whole percent). Useful for a final summary.
func (u *WorkerUtilization) UtilizedPercentWholeRun() int {
	n := len(u.hits)
	if n == 0 {
		return 0
	}
	active := 0
	for i := range u.hits {
		if atomic.LoadInt32(&u.hits[i]) > 0 {
			active++
		}
	}
	return int(math.Ceil(100.0 * float64(active) / float64(n)))
}
