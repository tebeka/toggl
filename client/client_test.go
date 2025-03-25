package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

type mockTripper struct {
	data   []byte
	status int
	err    error
}

// loadTestData loads a JSON test file from disk
// filename should be just the base filename (e.g., "projects.json")
func loadTestData(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "v8", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load test data %s: %v", path, err)
	}
	return data
}

func (mt mockTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if mt.err != nil {
		return nil, mt.err
	}

	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	if _, err := rec.Write(mt.data); err != nil {
		return nil, err
	}
	resp := rec.Result()
	if mt.status != 0 {
		resp.StatusCode = mt.status
	}
	return resp, nil
}

func newClient(t *testing.T) *Client {
	cfg := Config{
		APIToken:    "api-key",
		WorkspaceID: 1234,
		Timeout:     time.Second * 30,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				APIToken:    "token",
				WorkspaceID: 123,
				Timeout:     time.Second * 30,
			},
			expectError: false,
		},
		{
			name: "missing API token",
			config: Config{
				WorkspaceID: 123,
				Timeout:     time.Second * 30,
			},
			expectError: true,
		},
		{
			name: "missing workspace ID",
			config: Config{
				APIToken: "token",
				Timeout:  time.Second * 30,
			},
			expectError: true,
		},
		{
			name: "invalid timeout",
			config: Config{
				APIToken:    "token",
				WorkspaceID: 123,
				Timeout:     -time.Second,
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		APIToken:    "token",
		WorkspaceID: 123,
		Timeout:     time.Second * 30,
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected client but got nil")
	}

	if client.cfg != cfg {
		t.Errorf("expected config %v, got %v", cfg, client.cfg)
	}
}

func TestProjects(t *testing.T) {
	c := newClient(t)

	// First, we need to mock the Projects endpoint
	c.c.Transport = &mockTripper{data: loadTestData(t, "projects.json")}

	prjs, err := c.Projects()
	if err != nil {
		t.Fatal(err)
	}
	expected := []Project{
		{"A", 1, 0, ""},
		{"B", 2, 0, ""},
	}
	if !slices.Equal(prjs, expected) {
		t.Errorf("expected %v, got %v", expected, prjs)
	}
}

func TestClients(t *testing.T) {
	c := newClient(t)
	c.c.Transport = &mockTripper{data: loadTestData(t, "clients.json")}

	clients, err := c.Clients()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[int]string{
		101: "Client A",
		102: "Client B",
	}

	if len(clients) != len(expected) {
		t.Fatalf("expected %d clients, got %d", len(expected), len(clients))
	}

	for id, name := range expected {
		if clients[id] != name {
			t.Errorf("expected client %d to be %q, got %q", id, name, clients[id])
		}
	}
}

func TestProjectFullName(t *testing.T) {
	testCases := []struct {
		name     string
		project  Project
		expected string
	}{
		{
			name: "with client name",
			project: Project{
				Name:       "Project",
				ClientName: "Client",
			},
			expected: "Client/Project",
		},
		{
			name: "without client name",
			project: Project{
				Name: "Project",
			},
			expected: "Project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.project.FullName()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTimer(t *testing.T) {
	t.Run("timer running", func(t *testing.T) {
		c := newClient(t)
		c.c.Transport = &mockTripper{data: loadTestData(t, "timer.json")}

		timer, err := c.Timer()
		if err != nil {
			t.Fatal(err)
		}

		if timer == nil {
			t.Fatal("expected timer but got nil")
		}

		expectedID := 456
		if timer.ID != expectedID {
			t.Errorf("expected timer ID %d, got %d", expectedID, timer.ID)
		}

		expectedProject := 1
		if timer.Project != expectedProject {
			t.Errorf("expected project ID %d, got %d", expectedProject, timer.Project)
		}

		expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T14:30:45Z")
		if !timer.Start.Equal(expectedTime) {
			t.Errorf("expected start time %v, got %v", expectedTime, timer.Start)
		}
	})

	t.Run("no timer running", func(t *testing.T) {
		c := newClient(t)
		c.c.Transport = &mockTripper{data: loadTestData(t, "timer_empty.json")}

		timer, err := c.Timer()
		if err != nil {
			t.Fatal(err)
		}

		if timer != nil {
			t.Errorf("expected nil timer, got %+v", timer)
		}
	})
}

func TestStart(t *testing.T) {
	c := newClient(t)
	c.c.Transport = &mockTripper{data: loadTestData(t, "start_timer.json")}

	projectID := 123
	startTime := time.Now()

	err := c.Start(projectID, startTime)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStop(t *testing.T) {
	c := newClient(t)
	c.c.Transport = &mockTripper{data: loadTestData(t, "stop_timer.json")}

	timerID := 456
	projectID, duration, err := c.Stop(timerID)
	if err != nil {
		t.Fatal(err)
	}

	expectedProjectID := 1
	if projectID != expectedProjectID {
		t.Errorf("expected project ID %d, got %d", expectedProjectID, projectID)
	}

	expectedDuration := time.Hour
	if duration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, duration)
	}
}

func TestReport(t *testing.T) {
	c := newClient(t)
	c.c.Transport = &mockTripper{data: loadTestData(t, "report.json")}

	reports, err := c.Report("2023-01-01")
	if err != nil {
		t.Fatal(err)
	}

	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	expected := []Report{
		{Project: "Project A", Duration: time.Hour},
		{Project: "Project B", Duration: 2 * time.Hour},
	}

	for i, report := range reports {
		if report.Project != expected[i].Project {
			t.Errorf("expected project %q, got %q", expected[i].Project, report.Project)
		}

		if report.Duration != expected[i].Duration {
			t.Errorf("expected duration %v, got %v", expected[i].Duration, report.Duration)
		}
	}
}

func Test_callHTTPError(t *testing.T) {
	c := newClient(t)
	mt := mockTripper{status: http.StatusBadRequest}
	c.c.Transport = &mt

	err := c.call("GET", "https://go.dev", nil, nil)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func Test_timesURL(t *testing.T) {
	c := newClient(t)
	url := c.timesURL()
	expected := "https://api.track.toggl.com/api/v9/workspaces/1234/time_entries"

	if url != expected {
		t.Errorf("expected URL %q, got %q", expected, url)
	}
}
