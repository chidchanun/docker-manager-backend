package handler

import (
	"reflect"
	"testing"
)

func TestRedactCommandSecrets(t *testing.T) {
	input := []string{"tunnel", "run", "--token", "sensitive", "--api-key=also-sensitive", "--name", "safe"}
	want := []string{"tunnel", "run", "--token", "[REDACTED]", "--api-key=[REDACTED]", "--name", "safe"}
	if got := redactCommandSecrets(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if input[3] != "sensitive" {
		t.Fatal("input was mutated")
	}
}
