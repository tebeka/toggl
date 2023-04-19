package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	// APIBase is the base rest API URL
	APIBase = "https://api.track.toggl.com/api/v8"
)

var (
	baseURL = fmt.Sprintf("%s/time_entries", APIBase)
)

type Config struct {
	APIToken  string        `json:"api_token"`
	Workspace string        `json:"workspace"`
	Timeout   time.Duration `json:"timeout"`
}

type Client struct {
	apiToken  string
	workspace string
	timeout   time.Duration

	c http.Client
}

func New(cfg Config) (*Client, error) {
	c := &Client{
		apiToken:  cfg.APIToken,
		workspace: cfg.Workspace,
		timeout:   cfg.Timeout,
	}

	return c, nil
}

// call makes an API call with right credentials
func (c *Client) call(method, url string, body io.Reader, out interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.apiToken, "api_token")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %s %s - can't close body - %s", method, url, err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%q calling %s", resp.Status, url)
	}

	if out == nil {
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// Project is toggl project
type Project struct {
	Name       string `json:"name"`
	ID         int    `json:"id"`
	ClientID   int    `json:"cid"`
	ClientName string
}

func (p Project) FullName() string {
	if p.ClientName != "" {
		return fmt.Sprintf("%s/%s", p.ClientName, p.Name)
	}
	return p.Name
}

func (c *Client) Projects() ([]Project, error) {
	url := fmt.Sprintf("%s/workspaces/%s/projects", APIBase, c.workspace)
	var prjs []Project
	if err := c.call("GET", url, nil, &prjs); err != nil {
		return nil, err
	}

	clients, err := c.Clients()
	if err != nil {
		return nil, err
	}

	for i := range prjs {
		client := clients[prjs[i].ClientID]
		if client != "" {
			prjs[i].ClientName = client
		}
	}

	return prjs, nil
}

func (c *Client) Clients() (map[int]string, error) {
	url := fmt.Sprintf("%s/workspaces/%s/clients", APIBase, c.workspace)

	var cs []struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	if err := c.call("GET", url, nil, &cs); err != nil {
		return nil, err
	}

	ids := make(map[int]string) // id -> name
	for _, c := range cs {
		ids[c.ID] = c.Name
	}
	return ids, nil
}

// Timer is a toggle running timer
type Timer struct {
	ID      int       `json:"id"`
	Project int       `json:"pid"`
	Start   time.Time `json:"start"`
}

func (c *Client) Timer() (*Timer, error) {
	url := fmt.Sprintf("%s/current", baseURL)
	var reply struct {
		Data *Timer `json:"data"`
	}

	if err := c.call("GET", url, nil, &reply); err != nil {
		return nil, err
	}

	return reply.Data, nil
}

func (c *Client) Start(pid int) error {
	data := map[string]interface{}{
		"time_entry": map[string]interface{}{
			"pid":          pid,
			"description":  "",
			"created_with": "toggl",
		},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/start", baseURL)
	return c.call("POST", url, &buf, nil)
}

func (c *Client) Stop(id int) (int, time.Duration, error) {
	url := fmt.Sprintf("%s/%d/stop", baseURL, id)
	var reply struct {
		Data struct {
			Duration int `json:"duration"`
			ID       int `json:"pid"`
		}
	}
	if err := c.call("PUT", url, nil, &reply); err != nil {
		return -1, 0, err
	}

	dur := time.Duration(time.Duration(reply.Data.Duration) * time.Second)
	return reply.Data.ID, dur, nil
}

type Report struct {
	Project  string
	Duration time.Duration
}

func (c *Client) Report(since string) ([]Report, error) {
	u, err := url.Parse("https://api.track.toggl.com/reports/api/v2/summary")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("since", since)
	q.Set("workspace_id", c.workspace)
	q.Set("user_agent", "toggl")
	u.RawQuery = q.Encode()

	var reply struct {
		Data []struct {
			Title struct {
				Project string `json:"project"`
			} `json:"title"`
			Time int `json:"time"`
		} `json:"data"`
	}

	if err := c.call("GET", u.String(), nil, &reply); err != nil {
		return nil, err
	}

	var reports []Report
	for _, project := range reply.Data {
		d := time.Millisecond * time.Duration(project.Time)
		reports = append(reports, Report{project.Title.Project, d})
	}

	return reports, nil
}
