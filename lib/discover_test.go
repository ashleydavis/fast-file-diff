package lib

import (
	"testing"
)

func TestDiscoveredSet_addBothFormsPair(t *testing.T) {
	pool := NewPathPool()
	s := NewDiscoveredSet(pool)
	rel := "a/file.txt"
	if s.Add(rel, SideLeft) {
		t.Error("Add(rel, left) should not form pair yet")
	}
	if !s.Add(rel, SideRight) {
		t.Error("Add(rel, right) should form pair (left already seen)")
	}
	if s.Add(rel, SideLeft) {
		t.Error("Add(rel, left) again should not form new pair")
	}
}

func TestDiscoveredSet_leftOnlyNoPair(t *testing.T) {
	pool := NewPathPool()
	s := NewDiscoveredSet(pool)
	if s.Add("only/left", SideLeft) {
		t.Error("left-only path should not form pair")
	}
	if s.Add("only/left", SideLeft) {
		t.Error("still no pair")
	}
}

func TestDiscoveredSet_rightOnlyNoPair(t *testing.T) {
	pool := NewPathPool()
	s := NewDiscoveredSet(pool)
	if s.Add("only/right", SideRight) {
		t.Error("right-only path should not form pair")
	}
}

func TestDiscoveredSet_bothSidesNoOnly(t *testing.T) {
	pool := NewPathPool()
	s := NewDiscoveredSet(pool)
	s.Add("f", SideLeft)
	s.Add("f", SideRight)
	if len(s.LeftOnlyPaths()) != 0 {
		t.Errorf("LeftOnlyPaths() should be empty when both have f, got %v", s.LeftOnlyPaths())
	}
	if len(s.RightOnlyPaths()) != 0 {
		t.Errorf("RightOnlyPaths() should be empty when both have f, got %v", s.RightOnlyPaths())
	}
}

func TestDiscoveredSet_multiplePairs(t *testing.T) {
	pool := NewPathPool()
	s := NewDiscoveredSet(pool)
	for _, rel := range []string{"a", "b", "c"} {
		s.Add(rel, SideLeft)
	}
	pairs := 0
	for _, rel := range []string{"a", "b", "c"} {
		if s.Add(rel, SideRight) {
			pairs++
		}
	}
	if pairs != 3 {
		t.Errorf("expected 3 pairs, got %d", pairs)
	}
}
