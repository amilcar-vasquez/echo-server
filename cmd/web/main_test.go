// Filename: cmd/web/main_test.go

package main

import (
	"testing"
)

func TestMain(t *testing.T) {
	want := "Hello, RFC6455!"
	got := printWebSockets()

	if got != want {
		t.Errorf("expected: %q, got: %q", want, got)
	}
}