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
	data []byte
	err  error
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
	return rec.Result(), nil
}

func TestProjects(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		APIToken:  "api-key",
		Workspace: "workspace",
	}
	c, err := New(cfg)
	require.NoError(err)

	mt := mockTripper{projectsJSON, nil}
	c.c.Transport = &mt

	prjs, err := c.Projects()
	require.NoError(err)
	expected := []Project{
		{"A", 1, 0, ""},
		{"B", 2, 0, ""},
	}
	require.Equal(expected, prjs)
}
