package lib

import "sync/atomic"

// ProgressCounts holds counters and start time for the progress indicator.
// Exported fields so main can use atomic load for progress display.
// TotalPairs is the total number of pairs to compare (set before starting workers); 0 means unknown.
// If WorkerProcessed is non-nil and len(WorkerProcessed) >= numWorkers, ProgressRecorder increments WorkerProcessed[workerIdx] per completion so you can see per-worker counts (e.g. min/max in summary).
type ProgressCounts struct {
	Enqueued          int32
	Processed         int32
	StartTimeUnixNano int64
	TotalPairs        int32
	WorkerProcessed   []int32 // optional: per-worker compare count, one per worker
}

// ProgressRecorder records worker completions for the compare phase: it bumps Processed,
// optionally WorkerProcessed[workerIdx], and pokes the utilization tracker. Safe for concurrent use.
type ProgressRecorder struct {
	progress   *ProgressCounts
	utilization *WorkerUtilization
}

// NewProgressRecorder returns a recorder that updates progress and workerUtilization on each RecordCompletion.
// progress and workerUtilization must be non-nil.
func NewProgressRecorder(progress *ProgressCounts, workerUtilization *WorkerUtilization) *ProgressRecorder {
	return &ProgressRecorder{progress: progress, utilization: workerUtilization}
}

// RecordCompletion records that the given worker completed one unit of work (e.g. one file-pair comparison).
// It increments Processed, WorkerProcessed[workerIdx] when in range, and calls workerUtilization.Poke(workerIdx).
func (r *ProgressRecorder) RecordCompletion(workerIdx int) {
	atomic.AddInt32(&r.progress.Processed, 1)
	if workerIdx >= 0 && workerIdx < len(r.progress.WorkerProcessed) {
		atomic.AddInt32(&r.progress.WorkerProcessed[workerIdx], 1)
	}
	r.utilization.Poke(workerIdx)
}
