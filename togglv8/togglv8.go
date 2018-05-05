package togglv8

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// APIBase is the base rest API URL
	APIBase = "https://www.toggl.com/api/v8"
	// Version is current version
	Version  = "0.1.2"
	rcEnvKey = "TOGGLRC"
)

// Project is a toggl project.
type Project struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// TimeEntry describes a started time entry.
type TimeEntry struct {
	ID          int       `json:"id"`
	PID         int       `json:"pid"`
	WID         int       `json:"wid"`
	Start       time.Time `json:"start"`
	Stop        time.Time `json:"stop,omitempty"`
	Duration    int       `json:"duration,omitempty"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Billable    bool      `json:"billable"`
	At          time.Time `json:"at,omitempty"`
}

// Client is a client for fetching time reports etc from toggl.
type Client struct {
	workspaceID string
	apiToken    string

	*http.Client
}

// New creates a new client with a default HTTP client and values set.
func New(apiToken, workspaceID string) *Client {
	return &Client{
		apiToken:    apiToken,
		workspaceID: workspaceID,

		Client: http.DefaultClient,
	}
}

// Projects requests all projects for the given workspace.
func (c *Client) Projects() ([]Project, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/workspaces/%s/projects", APIBase, c.workspaceID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var ps []Project
	err = json.NewDecoder(resp.Body).Decode(&ps)
	return ps, err
}

// CurrentTimer returns the current running timer.
func (c *Client) CurrentTimer() (*TimeEntry, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/time_entries/current", APIBase), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Data *TimeEntry `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response.Data, err
}

// StartTimer starts a new timer.
func (c *Client) StartTimer(pid int, startTime time.Time, stopTime *time.Time, description string) (*TimeEntry, error) {
	var request struct {
		TimeEntry struct {
			PID         int    `json:"pid"`
			Description string `json:"description"`
			Start       string `json:"start"`
			Stop        string `json:"stop"`
			CreatedWith string `json:"created_with"`
			Duration    int    `json:"duration"`
			At          string `json:"at"`
		} `json:"time_entry"`
	}

	request.TimeEntry.PID = pid
	request.TimeEntry.Description = description
	request.TimeEntry.CreatedWith = "toggl"

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/time_entries/start", APIBase), &buf)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data *TimeEntry `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Always update it with data since starting doesn't seem to take all
	// parameters into consideration.
	return c.UpdateTimer(response.Data.ID, pid, startTime, stopTime, description)
}

// UpdateTimer updates an already running timer.
func (c *Client) UpdateTimer(id, pid int, startTime time.Time, stopTime *time.Time, description string) (*TimeEntry, error) {
	var request struct {
		TimeEntry struct {
			PID         int    `json:"pid"`
			Description string `json:"description"`
			Start       string `json:"start"`
			Stop        string `json:"stop"`
			CreatedWith string `json:"created_with"`
			Duration    int    `json:"duration"`
			At          string `json:"at"`
		} `json:"time_entry"`
	}

	request.TimeEntry.PID = pid
	request.TimeEntry.Description = description
	request.TimeEntry.Start = startTime.Format(time.RFC3339)
	request.TimeEntry.CreatedWith = "toggl"
	request.TimeEntry.At = time.Now().Format(time.RFC3339)

	if stopTime != nil {
		request.TimeEntry.Stop = stopTime.Format(time.RFC3339)
		request.TimeEntry.Duration = int(stopTime.Unix() - startTime.Unix())
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/time_entries/%d", APIBase, id), &buf)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data *TimeEntry `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Data, err
}

// StopTimer starts a new timer.
func (c *Client) StopTimer(id int) (time.Duration, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/time_entries/%d/stop", APIBase, id), nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var response struct {
		Data struct {
			Duration int `json:"duration"`
		}
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}

	return time.Duration(time.Duration(response.Data.Duration) * time.Second), nil
}

// Timers lists time entries, defaults to date range of 9 days.
func (c *Client) Timers() ([]TimeEntry, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/time_entries", APIBase), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []TimeEntry
	err = json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

// Do makes a request with headers set with the HTTP client specified in
// togglv8.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.apiToken, "api_token")
	req.Header.Set("Content-Type", "application/json")

	return c.Client.Do(req)
}
