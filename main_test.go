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

	c, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if c.Workspace != "123456" {
		t.Fatal("bad workspace")
	}
}
