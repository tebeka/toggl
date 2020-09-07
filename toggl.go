package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	// APIBase is the base rest API URL
	APIBase = "https://www.toggl.com/api/v8"
	// Version is current version
	Version  = "0.1.9"
	rcEnvKey = "TOGGLRC"

	usage = `usage: %s start <project>|stop|status|projects|report <since>
	<project> - project name
	<since> - YYYY-MM-DD (default to start of today)
`
)

var (
	baseURL = fmt.Sprintf("%s/time_entries", APIBase)
	config  struct {
		APIToken  string `json:"api_token"`
		Workspace string `json:"workspace"`
	}
	unknownProject = "<unknown>"
)

// Project is toggl project
type Project struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// Timer is a toggle running timer
type Timer struct {
	ID      int       `json:"id"`
	Project int       `json:"pid"`
	Start   time.Time `json:"start"`
}

func configFile() (string, error) {
	path := os.Getenv(rcEnvKey)
	if len(path) > 0 {
		return path, nil
	}
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/.togglrc", user.HomeDir), nil
}

func loadConfig() error {
	fname, err := configFile()
	if err != nil {
		return err
	}

	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	return dec.Decode(&config)
}

// APICall makes an API call with right credentials
func APICall(method, url string, body io.Reader, out interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth(config.APIToken, "api_token")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%q calling %s", resp.Status, url)
	}

	if out == nil {
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

func getProjects() ([]Project, error) {
	url := fmt.Sprintf("%s/workspaces/%s/projects", APIBase, config.Workspace)
	var prjs []Project
	if err := APICall("GET", url, nil, &prjs); err != nil {
		return nil, err
	}

	return prjs, nil
}

func printProjects(prjs []Project) {
	var names []string
	for _, prj := range prjs {
		names = append(names, prj.Name)
	}

	cmp := func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	}

	sort.Slice(names, cmp)
	for _, name := range names {
		fmt.Println(name)
	}
}

func currentTimer() (*Timer, error) {
	url := fmt.Sprintf("%s/current", baseURL)
	var reply struct {
		Data *Timer `json:"data"`
	}

	if err := APICall("GET", url, nil, &reply); err != nil {
		return nil, err
	}

	return reply.Data, nil
}

func findProject(name string, prjs []Project) []Project {
	var matches []Project
	name = strings.ToLower(name)
	for _, prj := range prjs {
		if strings.HasPrefix(strings.ToLower(prj.Name), name) {
			matches = append(matches, prj)
		}
	}
	return matches
}

func nameFromID(id int, prjs []Project) string {
	for _, prj := range prjs {
		if prj.ID == id {
			return prj.Name
		}
	}

	return ""
}

func checkCommand(command string) error {
	switch command {
	case "start":
		if flag.NArg() != 2 {
			return fmt.Errorf("wrong number of arguments")
		}
	case "stop", "status", "projects":
		if flag.NArg() != 1 {
			return fmt.Errorf("wrong number of arguments")
		}
	case "report":
		if flag.NArg() > 2 {
			return fmt.Errorf("wrong number of arguments")
		}
	default:
		return fmt.Errorf("unknown command - %s", flag.Arg(0))
	}
	return nil
}

func duration2str(dur time.Duration) string {
	h, m, s := int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func startTimer(pid int) error {
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
	return APICall("POST", url, &buf, nil)
}

func stopTimer(id int) (int, time.Duration, error) {
	url := fmt.Sprintf("%s/%d/stop", baseURL, id)
	var reply struct {
		Data struct {
			Duration int `json:"duration"`
			ID       int `json:"pid"`
		}
	}
	if err := APICall("PUT", url, nil, &reply); err != nil {
		return -1, 0, err
	}

	dur := time.Duration(time.Duration(reply.Data.Duration) * time.Second)
	return reply.Data.ID, dur, nil
}

func findCmd(prefix string) []string {
	commands := []string{"start", "stop", "status", "projects", "report"}
	var matches []string

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	return matches
}

func report(since string) error {
	u, err := url.Parse("https://toggl.com/reports/api/v2/summary")
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("since", since)
	q.Set("workspace_id", config.Workspace)
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

	if err := APICall("GET", u.String(), nil, &reply); err != nil {
		return err
	}

	for _, project := range reply.Data {
		d := time.Millisecond * time.Duration(project.Time)
		fmt.Printf("%s: %s\n", project.Title.Project, d)
	}

	return nil
}

func main() {
	log.SetFlags(0) // Don't prefix with time
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "show version and exit")
	flag.Usage = func() {
		name := path.Base(os.Args[0])
		fmt.Printf(usage, name)
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		fmt.Printf("%s\n", Version)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		log.Fatalf("error: wrong number of arguments")
	}

	matches := findCmd(flag.Arg(0))
	switch len(matches) {
	case 0:
		log.Fatalf("error: unknown command - %q", flag.Arg(0))
	case 1: /* nop */
	default:
		log.Fatalf("error: too many matches to %q", flag.Arg(0))
	}

	command := matches[0]
	if err := checkCommand(command); err != nil {
		log.Fatalf("error: %s", err)
	}

	if err := loadConfig(); err != nil {
		log.Fatalf("error: can't load config - %s", err)
	}
	prjs, err := getProjects()
	if err != nil {
		log.Fatalf("error: can't get projects - %s", err)
	}

	curTimer, err := currentTimer()
	if err != nil {
		log.Fatalf("error: can't get current timer - %s", err)
	}

	switch command {
	case "projects":
		printProjects(prjs)
	case "start":
		if curTimer != nil {
			name := nameFromID(curTimer.Project, prjs)
			if name == "" {
				name = unknownProject
			}
			log.Fatalf("error: there is a timer running for %q", name)
		}
		name := flag.Arg(1)
		matches := findProject(name, prjs)
		switch len(matches) {
		case 0:
			log.Fatalf("error: no project match %s", name)
		case 1:
		default:
			log.Fatalf("error: too project many matches to %s", name)
		}
		fmt.Printf("Starting %s\n", matches[0].Name)
		if err := startTimer(matches[0].ID); err != nil {
			log.Fatalf("error: can't start timer - %s", err)
		}
	case "stop":
		if curTimer == nil {
			log.Fatalf("error: no timer running")
		}
		pid, dur, err := stopTimer(curTimer.ID)
		if err != nil {
			log.Fatalf("error: can't stop timer - %s", err)
		}
		name := nameFromID(pid, prjs)
		if name == "" {
			name = unknownProject
		}
		fmt.Printf("%s: %s\n", name, duration2str(dur))
	case "status":
		if curTimer == nil {
			fmt.Println("no timer is running")
			return
		}
		name := nameFromID(curTimer.Project, prjs)
		if name == "" {
			name = unknownProject
		}
		dur := time.Since(curTimer.Start)
		fmt.Printf("%s: %s\n", name, duration2str(dur))
	case "report":
		yday := time.Now().Add(-24 * time.Hour)
		since := yday.Format("2006-01-02")
		if flag.NArg() == 2 {
			since = flag.Arg(1)
		}

		if err := report(since); err != nil {
			log.Fatalf("error: can't get report - %s", err)
		}
	}
}
