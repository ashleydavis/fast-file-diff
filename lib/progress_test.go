package lib

import (
	"sync/atomic"
	"testing"
)

func TestNewProgressRecorder(t *testing.T) {
	progress := &ProgressCounts{WorkerProcessed: make([]int32, 2)}
	util := NewWorkerUtilization(2, 10)
	rec := NewProgressRecorder(progress, util)
	if rec == nil {
		t.Fatal("NewProgressRecorder returned nil")
	}
}

func TestRecordCompletion_incrementsProcessed(t *testing.T) {
	progress := &ProgressCounts{WorkerProcessed: make([]int32, 2)}
	util := NewWorkerUtilization(2, 10)
	rec := NewProgressRecorder(progress, util)

	rec.RecordCompletion(0)
	if atomic.LoadInt32(&progress.Processed) != 1 {
		t.Errorf("Processed = %d, want 1", atomic.LoadInt32(&progress.Processed))
	}
	rec.RecordCompletion(1)
	rec.RecordCompletion(0)
	if atomic.LoadInt32(&progress.Processed) != 3 {
		t.Errorf("Processed = %d, want 3", atomic.LoadInt32(&progress.Processed))
	}
}

func TestRecordCompletion_incrementsWorkerProcessed(t *testing.T) {
	progress := &ProgressCounts{WorkerProcessed: make([]int32, 3)}
	util := NewWorkerUtilization(3, 10)
	rec := NewProgressRecorder(progress, util)

	rec.RecordCompletion(0)
	rec.RecordCompletion(0)
	rec.RecordCompletion(1)
	if atomic.LoadInt32(&progress.WorkerProcessed[0]) != 2 {
		t.Errorf("WorkerProcessed[0] = %d, want 2", atomic.LoadInt32(&progress.WorkerProcessed[0]))
	}
	if atomic.LoadInt32(&progress.WorkerProcessed[1]) != 1 {
		t.Errorf("WorkerProcessed[1] = %d, want 1", atomic.LoadInt32(&progress.WorkerProcessed[1]))
	}
	if atomic.LoadInt32(&progress.WorkerProcessed[2]) != 0 {
		t.Errorf("WorkerProcessed[2] = %d, want 0", atomic.LoadInt32(&progress.WorkerProcessed[2]))
	}
}

func TestRecordCompletion_workerIdxOutOfRange(t *testing.T) {
	progress := &ProgressCounts{WorkerProcessed: make([]int32, 2)}
	util := NewWorkerUtilization(2, 10)
	rec := NewProgressRecorder(progress, util)

	rec.RecordCompletion(-1)
	rec.RecordCompletion(2)
	rec.RecordCompletion(10)
	if atomic.LoadInt32(&progress.Processed) != 3 {
		t.Errorf("Processed = %d, want 3 (out-of-range still increments Processed)", atomic.LoadInt32(&progress.Processed))
	}
	if atomic.LoadInt32(&progress.WorkerProcessed[0]) != 0 || atomic.LoadInt32(&progress.WorkerProcessed[1]) != 0 {
		t.Errorf("WorkerProcessed should be unchanged for out-of-range idx")
	}
}

func TestRecordCompletion_pokesUtilization(t *testing.T) {
	progress := &ProgressCounts{WorkerProcessed: make([]int32, 2)}
	util := NewWorkerUtilization(2, 10)
	rec := NewProgressRecorder(progress, util)

	rec.RecordCompletion(0)
	rec.RecordCompletion(1)
	if util.UtilizedPercentWholeRun() != 100 {
		t.Errorf("UtilizedPercentWholeRun = %d%%, want 100%%", util.UtilizedPercentWholeRun())
	}
}

func TestRecordCompletion_concurrent(t *testing.T) {
	const workers = 4
	const perWorker = 100
	progress := &ProgressCounts{WorkerProcessed: make([]int32, workers)}
	util := NewWorkerUtilization(workers, 10)
	rec := NewProgressRecorder(progress, util)

	done := make(chan struct{})
	for w := 0; w < workers; w++ {
		idx := w
		go func() {
			for i := 0; i < perWorker; i++ {
				rec.RecordCompletion(idx)
			}
			done <- struct{}{}
		}()
	}
	for w := 0; w < workers; w++ {
		<-done
	}

	if atomic.LoadInt32(&progress.Processed) != workers*perWorker {
		t.Errorf("Processed = %d, want %d", atomic.LoadInt32(&progress.Processed), workers*perWorker)
	}
	for w := 0; w < workers; w++ {
		if atomic.LoadInt32(&progress.WorkerProcessed[w]) != perWorker {
			t.Errorf("WorkerProcessed[%d] = %d, want %d", w, atomic.LoadInt32(&progress.WorkerProcessed[w]), perWorker)
		}
	}
	if util.UtilizedPercentWholeRun() != 100 {
		t.Errorf("UtilizedPercentWholeRun = %d%%, want 100%%", util.UtilizedPercentWholeRun())
	}
}
