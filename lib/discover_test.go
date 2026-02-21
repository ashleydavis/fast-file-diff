package lib

import (
	"testing"
	"time"
)

func TestDiscoveredSet_addBothFormsPair(t *testing.T) {
	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	rel := "a/file.txt"
	if set.Add(rel, SideLeft, 0, time.Time{}) {
		t.Error("Add(rel, left) should not form pair yet")
	}
	if !set.Add(rel, SideRight, 0, time.Time{}) {
		t.Error("Add(rel, right) should form pair (left already seen)")
	}
	if set.Add(rel, SideLeft, 0, time.Time{}) {
		t.Error("Add(rel, left) again should not form new pair")
	}
}

func TestDiscoveredSet_leftOnlyNoPair(t *testing.T) {
	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	if set.Add("only/left", SideLeft, 0, time.Time{}) {
		t.Error("left-only path should not form pair")
	}
	if set.Add("only/left", SideLeft, 0, time.Time{}) {
		t.Error("still no pair")
	}
}

func TestDiscoveredSet_rightOnlyNoPair(t *testing.T) {
	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	if set.Add("only/right", SideRight, 0, time.Time{}) {
		t.Error("right-only path should not form pair")
	}
}

func TestDiscoveredSet_bothSidesNoOnly(t *testing.T) {
	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	set.Add("f", SideLeft, 0, time.Time{})
	set.Add("f", SideRight, 0, time.Time{})
	if len(set.LeftOnlyPaths()) != 0 {
		t.Errorf("LeftOnlyPaths() should be empty when both have f, got %v", set.LeftOnlyPaths())
	}
	if len(set.RightOnlyPaths()) != 0 {
		t.Errorf("RightOnlyPaths() should be empty when both have f, got %v", set.RightOnlyPaths())
	}
}

func TestDiscoveredSet_multiplePairs(t *testing.T) {
	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	for _, rel := range []string{"a", "b", "c"} {
		set.Add(rel, SideLeft, 0, time.Time{})
	}
	pairsCount := 0
	for _, rel := range []string{"a", "b", "c"} {
		if set.Add(rel, SideRight, 0, time.Time{}) {
			pairsCount++
		}
	}
	if pairsCount != 3 {
		t.Errorf("expected 3 pairs, got %d", pairsCount)
	}
}
