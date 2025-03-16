package main

import (
	"os"
	"testing"
	"time"

	"github.com/tebeka/toggl/client"
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

	expected := client.Config{
		APIToken:    "43c48580e5ad47fa820608eca77eb161",
		WorkspaceID: 123456,
		Timeout:     5 * time.Second,
	}

	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}
