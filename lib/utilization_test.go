package lib

import (
	"sync/atomic"
	"testing"
)

func TestNewWorkerUtilization_validArgs(t *testing.T) {
	u := NewWorkerUtilization(4, 10)
	if u == nil {
		t.Fatal("NewWorkerUtilization returned nil")
	}
	if len(u.hits) != 4 {
		t.Errorf("hits length = %d, want 4", len(u.hits))
	}
	if u.windowTicks != 10 {
		t.Errorf("windowTicks = %d, want 10", u.windowTicks)
	}
}

func TestNewWorkerUtilization_zeroWorkersUsesOne(t *testing.T) {
	u := NewWorkerUtilization(0, 5)
	if len(u.hits) != 1 {
		t.Errorf("hits length = %d, want 1", len(u.hits))
	}
}

func TestPoke_inRangeIncrements(t *testing.T) {
	u := NewWorkerUtilization(3, 10)
	u.Poke(0)
	u.Poke(0)
	u.Poke(1)
	if atomic.LoadInt32(&u.hits[0]) != 2 {
		t.Errorf("hits[0] = %d, want 2", atomic.LoadInt32(&u.hits[0]))
	}
	if atomic.LoadInt32(&u.hits[1]) != 1 {
		t.Errorf("hits[1] = %d, want 1", atomic.LoadInt32(&u.hits[1]))
	}
	if atomic.LoadInt32(&u.hits[2]) != 0 {
		t.Errorf("hits[2] = %d, want 0", atomic.LoadInt32(&u.hits[2]))
	}
}

func TestPoke_outOfRangeNoOp(t *testing.T) {
	u := NewWorkerUtilization(2, 10)
	u.Poke(-1)
	u.Poke(2)
	u.Poke(100)
	for i := range u.hits {
		if atomic.LoadInt32(&u.hits[i]) != 0 {
			t.Errorf("hits[%d] = %d, want 0", i, atomic.LoadInt32(&u.hits[i]))
		}
	}
}

func TestTick_emptyHistoryUsesCurrentNonZero(t *testing.T) {
	u := NewWorkerUtilization(4, 10)
	u.Poke(0)
	u.Poke(2)
	pct := u.Tick()
	// 2 of 4 workers have hit > 0 => 50%, ceil = 50
	if pct != 50 {
		t.Errorf("Tick() = %d%%, want 50%%", pct)
	}
}

func TestTick_afterWindowComparesToOldest(t *testing.T) {
	u := NewWorkerUtilization(2, 3)
	u.Tick() // history = [snap0: 0,0]
	u.Poke(0)
	u.Poke(0)
	u.Tick() // history = [snap0, snap1: 2,0]
	u.Poke(1)
	u.Tick() // history = [snap0, snap1, snap2: 2,1]
	u.Poke(0)
	// Tick: cur = [3,1], history = [snap1, snap2, snap3]; compare snap3 to snap1 => worker0 3>2, worker1 1>0 => both active => 100%
	pct := u.Tick()
	if pct != 100 {
		t.Errorf("Tick() = %d%%, want 100%%", pct)
	}
}

func TestTick_noHitsInWindowReturnsZero(t *testing.T) {
	u := NewWorkerUtilization(3, 2)
	u.Tick() // [all zeros]
	u.Tick() // [zeros, zeros]; current vs oldest => no change => 0% active
	pct := u.Tick()
	if pct != 0 {
		t.Errorf("Tick() = %d%%, want 0%%", pct)
	}
}

func TestUtilizedPercentWholeRun(t *testing.T) {
	u := NewWorkerUtilization(4, 10)
	if u.UtilizedPercentWholeRun() != 0 {
		t.Errorf("empty run = %d%%, want 0%%", u.UtilizedPercentWholeRun())
	}
	u.Poke(1)
	u.Poke(3)
	// 2 of 4 => 50%
	if u.UtilizedPercentWholeRun() != 50 {
		t.Errorf("two workers = %d%%, want 50%%", u.UtilizedPercentWholeRun())
	}
	u.Poke(0)
	u.Poke(2)
	// 4 of 4 => 100%
	if u.UtilizedPercentWholeRun() != 100 {
		t.Errorf("all workers = %d%%, want 100%%", u.UtilizedPercentWholeRun())
	}
}

func TestUtilizedPercentWholeRun_singleWorker(t *testing.T) {
	u := NewWorkerUtilization(1, 10)
	u.Poke(0)
	if u.UtilizedPercentWholeRun() != 100 {
		t.Errorf("one worker did work = %d%%, want 100%%", u.UtilizedPercentWholeRun())
	}
}
