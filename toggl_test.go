package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	oldVal := os.Getenv(rcEnvKey)
	defer func() {
		os.Setenv(rcEnvKey, oldVal)
	}()

	os.Setenv(rcEnvKey, "togglrc-example")

	var c config
	if err := loadConfig(&c); err != nil {
		t.Fatal(err)
	}

	if c.Workspace != "123456" {
		t.Fatal("bad workspace")
	}
}

func TestFindCmd(t *testing.T) {
	testCases := []struct {
		cmd string
		n   int
	}{
		{"st", 3}, // start, stop, status
		{"x", 0},
		{"sta", 2},  // start, status
		{"stat", 1}, // status
	}

	for _, tc := range testCases {
		t.Run(tc.cmd, func(t *testing.T) {
			matches := findCmd(tc.cmd)
			if len(matches) != tc.n {
				t.Fatalf("bad matches for %v - %v, (expected %d)", tc.cmd, matches, tc.n)
			}
		})
	}

}
