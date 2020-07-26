package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

var (
	zeroTime = time.Time{}
	start    = zeroTime
	apiKey   = "s3cr3t"
	projects = []map[string]interface{}{
		{"name": "p1", "id": 1},
		{"name": "p2", "id": 2},
	}
)

type Auth struct{}

func (a Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, passwd, ok := r.BasicAuth()
	if !ok || user == "" || passwd == "" {
		http.Error(w, "No auth information", http.StatusUnauthorized)
		return
	}

	if !ok || user != apiKey {
		http.Error(w, "Bad auth information", http.StatusUnauthorized)
		return
	}

	http.DefaultServeMux.ServeHTTP(w, r)
}

func projectsHandler(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.Encode(projects)

}

func startHTTPD(t *testing.T) string {
	port := freePort(t)
	http.HandleFunc("/api/v8/workspaces/3/projects", projectsHandler)
	addr := fmt.Sprintf(":%d", port)
	go http.ListenAndServe(addr, Auth{})

	waitForServer(t, port)
	return fmt.Sprintf("http://localhost:%d", port)
}

func freePort(t *testing.T) int {
	conn, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatalf("can't find free port - %s", err)
	}

	conn.Close()
	return conn.Addr().(*net.TCPAddr).Port
}

func waitForServer(t *testing.T, port int) {
	addr := fmt.Sprintf("localhost:%d", port)
	start := time.Now()
	timeout := 10 * time.Second
	var err error
	for time.Now().Sub(start) < timeout {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("server not ready after %s (%s)", timeout, err)
}
