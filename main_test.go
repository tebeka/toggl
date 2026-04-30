package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func TestResolveStartProject(t *testing.T) {
	projects := []client.Project{
		{ID: 1, Name: "cartwheel"},
		{ID: 2, Name: "jump"},
		{ID: 3, Name: "wheel"},
	}

	t.Run("single match", func(t *testing.T) {
		prj, err := resolveStartProject([]string{"jmp"}, projects, false, nil)
		if err != nil {
			t.Fatal(err)
		}

		if prj != projects[1] {
			t.Fatalf("expected %#v, got %#v", projects[1], prj)
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, err := resolveStartProject([]string{"banana"}, projects, false, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := "error: no project match banana"
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err)
		}
	})

	t.Run("multiple matches", func(t *testing.T) {
		_, err := resolveStartProject([]string{"whl"}, projects, false, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := `too many matches to "whl": cartwheel, wheel`
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err)
		}
	})

	t.Run("interactive chooser", func(t *testing.T) {
		called := false
		prj, err := resolveStartProject(nil, projects, true, func(got []client.Project) (client.Project, error) {
			called = true
			if !slices.Equal(got, projects) {
				t.Fatalf("expected %#v, got %#v", projects, got)
			}

			return projects[2], nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !called {
			t.Fatal("chooser was not called")
		}

		if prj != projects[2] {
			t.Fatalf("expected %#v, got %#v", projects[2], prj)
		}
	})

	t.Run("chooser error", func(t *testing.T) {
		expected := errors.New("selection failed")
		_, err := resolveStartProject(nil, projects, true, func([]client.Project) (client.Project, error) {
			return client.Project{}, expected
		})
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	})

	t.Run("non interactive needs argument", func(t *testing.T) {
		_, err := resolveStartProject(nil, projects, false, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := "wrong number of arguments"
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err)
		}
	})
}

func TestBadReportDate(t *testing.T) {
	dir := t.TempDir()
	exe := fmt.Sprintf("%s/%s", dir, "toggl")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command(exe, "report", "01-02-2023")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
