package main

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRequireZeroOrTwoArgs(t *testing.T) {
	cmd := &cobra.Command{}
	if err := requireZeroOrTwoArgs(cmd, nil); err != nil {
		t.Errorf("requireZeroOrTwoArgs(nil) = %v", err)
	}
	if err := requireZeroOrTwoArgs(cmd, []string{"a", "b"}); err != nil {
		t.Errorf("requireZeroOrTwoArgs([a,b]) = %v", err)
	}
	if err := requireZeroOrTwoArgs(cmd, []string{"only"}); err == nil {
		t.Error("requireZeroOrTwoArgs([one]) want error")
	}
	if err := requireZeroOrTwoArgs(cmd, []string{"a", "b", "c"}); err == nil {
		t.Error("requireZeroOrTwoArgs([a,b,c]) want error")
	}
}

func TestEstimateRemainingFromElapsed(t *testing.T) {
	elapsed := 10 * time.Second
	// 10 processed in 10s => 1s per pair; 5 pending => 5s remaining
	got := estimateRemainingFromElapsed(elapsed, 10, 5)
	if got != 5*time.Second {
		t.Errorf("estimateRemainingFromElapsed(10s, 10, 5) = %v, want 5s", got)
	}
	// processed 0 => no estimate
	got = estimateRemainingFromElapsed(elapsed, 0, 5)
	if got != 0 {
		t.Errorf("estimateRemainingFromElapsed(10s, 0, 5) = %v, want 0", got)
	}
	// pending 0 => 0 remaining
	got = estimateRemainingFromElapsed(elapsed, 10, 0)
	if got != 0 {
		t.Errorf("estimateRemainingFromElapsed(10s, 10, 0) = %v, want 0", got)
	}
}
