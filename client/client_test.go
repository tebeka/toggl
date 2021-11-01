package client

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed "testdata/projects.json"
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
	if _, err := rec.Write(mt.data); err != nil {
		return nil, err
	}
	rec.Header().Set("Content-Type", "application/json")
	return rec.Result(), nil
}

func TestProjects(t *testing.T) {
	require := require.New(t)
	c, err := New("api-key", "workspace")
	require.NoError(err)

	mt := mockTripper{projectsJSON, nil}
	c.c.Transport = &mt

	prjs, err := c.Projects()
	require.NoError(err)
	expected := []Project{
		{"A", 1, 0},
		{"B", 2, 0},
	}
	require.Equal(expected, prjs)
}
