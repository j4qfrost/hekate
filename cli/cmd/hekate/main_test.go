package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(stdout.String(), "hekate") {
		t.Errorf("version output missing program name: %q", stdout.String())
	}
}

func TestUsageOnNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(nil, &stdout, &stderr); err != nil {
		t.Fatalf("no-args: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage text on stdout: %q", stdout.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"frobnicate"}, &stdout, &stderr); err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestVenueNotImplementedYet(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"venue", "create"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected not-implemented error from venue create, got %v", err)
	}
}
