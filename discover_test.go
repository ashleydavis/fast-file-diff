package main

import (
	"testing"
)

func TestDiscoveredSet_addBothFormsPair(t *testing.T) {
	pool := newPathPool()
	s := newDiscoveredSet(pool)
	rel := "a/file.txt"
	if s.Add(rel, sideLeft) {
		t.Error("Add(rel, left) should not form pair yet")
	}
	if !s.Add(rel, sideRight) {
		t.Error("Add(rel, right) should form pair (left already seen)")
	}
	// Adding again should not form a new pair
	if s.Add(rel, sideLeft) {
		t.Error("Add(rel, left) again should not form new pair")
	}
}

func TestDiscoveredSet_leftOnlyNoPair(t *testing.T) {
	pool := newPathPool()
	s := newDiscoveredSet(pool)
	if s.Add("only/left", sideLeft) {
		t.Error("left-only path should not form pair")
	}
	if s.Add("only/left", sideLeft) {
		t.Error("still no pair")
	}
}

func TestDiscoveredSet_rightOnlyNoPair(t *testing.T) {
	pool := newPathPool()
	s := newDiscoveredSet(pool)
	if s.Add("only/right", sideRight) {
		t.Error("right-only path should not form pair")
	}
}

func TestDiscoveredSet_multiplePairs(t *testing.T) {
	pool := newPathPool()
	s := newDiscoveredSet(pool)
	for _, rel := range []string{"a", "b", "c"} {
		s.Add(rel, sideLeft)
	}
	pairs := 0
	for _, rel := range []string{"a", "b", "c"} {
		if s.Add(rel, sideRight) {
			pairs++
		}
	}
	if pairs != 3 {
		t.Errorf("expected 3 pairs, got %d", pairs)
	}
}
