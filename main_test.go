package main

import (
	"os"
	"slices"
	"sort"
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

func Test_findProject(t *testing.T) {
	projects := []client.Project{
		{ID: 1, Name: "cartwheel"},
		{ID: 2, Name: "jump"},
		{ID: 3, Name: "wheel"},
		{ID: 4, Name: "walk"},
	}

	cases := []struct {
		query    string
		expected []client.Project
	}{
		{"whl", []client.Project{projects[0], projects[2]}},
		{"jmp", []client.Project{projects[1]}},
		{"banana", nil},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			found := findProject(tc.query, projects)
			sort.Slice(found, func(i, j int) bool {
				return found[i].ID < found[j].ID
			})

			if !slices.Equal(found, tc.expected) {
				t.Errorf("expected %#v, got %#v", tc.expected, found)
			}
		})
	}
}
