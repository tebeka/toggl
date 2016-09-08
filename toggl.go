package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
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
)

var (
	baseURL = fmt.Sprintf("%s/time_entries", APIBase)
	config  struct {
		APIToken  string `json:"api_token"`
		Workspace string `json:"workspace"`
	}
)

// Project is toggl project
type Project struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// Timer is a toggle running timer
type Timer struct {
	ID    int       `json:"id"`
	Start time.Time `json:"start"`
}

func loadConfig() error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	fname := fmt.Sprintf("%s/.togglrc", user.HomeDir)
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	if err := dec.Decode(&config); err != nil {
		return err
	}

	return nil
}

// APICall makes an API call with right credentials
func APICall(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(config.APIToken, "api_token")
	return http.DefaultClient.Do(req)
}

func getProjects() ([]Project, error) {
	url := fmt.Sprintf("%s/workspaces/%s/projects", APIBase, config.Workspace)
	resp, err := APICall("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var prjs []Project
	if err := dec.Decode(&prjs); err != nil {
		return nil, err
	}
	return prjs, nil
}

func printProjects(prjs []Project) {
	var names []string
	for _, prj := range prjs {
		names = append(names, prj.Name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Println(name)
	}
}

func currentTimer() (*Timer, error) {
	url := fmt.Sprintf("%s/current", baseURL)
	resp, err := APICall("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reply struct {
		Data *Timer `json:"data"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&reply); err != nil {
		return nil, err
	}

	return reply.Data, nil
}

func findProject(name string, prjs []Project) []int {
	var matches []int
	name = strings.ToLower(name)
	for _, prj := range prjs {
		if strings.HasPrefix(strings.ToLower(prj.Name), name) {
			matches = append(matches, prj.ID)
		}
	}
	return matches
}

func checkArgs() error {
	switch flag.Arg(0) {
	case "start":
		if flag.NArg() != 2 {
			return fmt.Errorf("wrong number of arguments")
		}
	case "stop", "status", "projects":
		if flag.NArg() != 1 {
			return fmt.Errorf("wrong number of arguments")
		}
	default:
		return fmt.Errorf("unknown command - %s", flag.Arg(0))
	}
	return nil
}

func duration2str(dur time.Duration) string {
	h, m, s := int(dur.Hours()), int(dur.Minutes()/60.0), int(dur.Seconds())%60
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
	if _, err := APICall("POST", url, &buf); err != nil {
		return err
	}
	return nil
}

func stopTimer(id int) (time.Duration, error) {
	url := fmt.Sprintf("%s/%d/stop", baseURL, id)
	resp, err := APICall("PUT", url, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var reply struct {
		Data struct {
			Duration int `json:"duration"`
		}
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&reply); err != nil {
		return 0, err
	}
	dur := time.Duration(time.Duration(reply.Data.Duration) * time.Second)
	return dur, err
}

func main() {
	log.SetFlags(0) // Don't prefix with time
	flag.Usage = func() {
		name := path.Base(os.Args[0])
		fmt.Printf("usage: %s [start <project>|stop|status|projects]\n", name)
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := checkArgs(); err != nil {
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

	switch flag.Arg(0) {
	case "projects":
		printProjects(prjs)
	case "start":
		if curTimer != nil {
			log.Fatalf("error: there is a timer running")
		}
		name := flag.Arg(1)
		ids := findProject(name, prjs)
		switch len(ids) {
		case 0:
			log.Fatalf("error: no project match %s", name)
		case 1:
		default:
			log.Fatalf("error: too project many matches to %s", name)
		}
		if err := startTimer(ids[0]); err != nil {
			log.Fatalf("error: can't start timer - %s", err)
		}
	case "stop":
		if curTimer == nil {
			log.Fatalf("error: no timer running")
		}
		dur, err := stopTimer(curTimer.ID)
		if err != nil {
			log.Fatalf("error: can't stop timer - %s", err)
		}
		fmt.Printf("%s\n", duration2str(dur))
	case "status":
		if curTimer == nil {
			log.Fatalf("error: no timer running")
		}
		dur := time.Now().UTC().Sub(curTimer.Start)
		fmt.Printf("duration: %s\n", duration2str(dur))
	}
}
