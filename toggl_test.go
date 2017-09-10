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

	if err := loadConfig(); err != nil {
		t.Fatal(err)
	}

	if config.Workspace != "123456" {
		t.Fatal("bad workspace")
	}
}
