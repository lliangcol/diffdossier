package main

import (
	"strings"
	"testing"
)

func TestCompatibilityPathNamesRespectOperatingSystemRules(t *testing.T) {
	windows := compatibilityPathNames("windows")
	for _, name := range windows {
		if strings.ContainsAny(name, "\t\n\r") {
			t.Fatalf("Windows fixture contains a forbidden control character: %q", name)
		}
	}
	unix := compatibilityPathNames("linux")
	if len(unix) != len(windows)+2 {
		t.Fatalf("Unix fixture has %d paths, want %d", len(unix), len(windows)+2)
	}
	if !containsName(unix, "tab\tname.txt") || !containsName(unix, "line\nname.txt") {
		t.Fatal("Unix fixture must retain tab and newline path coverage")
	}
}

func containsName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
