package client

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed "testdata/v8/projects.json"
	projectsJSON []byte
)

type mockTripper struct {
	data   []byte
	status int
	err    error
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
	}
	c, err := New(cfg)
	require.NoError(t, err)
	return c
}

func TestProjects(t *testing.T) {
	c := newClient(t)

	mt := mockTripper{data: projectsJSON}
	c.c.Transport = &mt

	prjs, err := c.Projects()
	require.NoError(t, err)
	expected := []Project{
		{"A", 1, 0, ""},
		{"B", 2, 0, ""},
	}
	require.Equal(t, expected, prjs)
}

func Test_callHTTPError(t *testing.T) {
	c := newClient(t)
	mt := mockTripper{status: http.StatusBadRequest}
	c.c.Transport = &mt

	err := c.call("GET", "https://go.dev", nil, nil)
	require.Error(t, err)
}
