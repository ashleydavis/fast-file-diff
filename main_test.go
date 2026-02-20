package main

import "testing"

func TestHelloWorld(t *testing.T) {
	got := "hello world"
	want := "hello world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
