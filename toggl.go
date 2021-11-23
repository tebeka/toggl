package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/tebeka/toggl/client"
)

const (
	// Version is current version
	Version  = "0.3.1"
	rcEnvKey = "TOGGLRC"

	usage = `usage: %s start <project>|stop|status|projects|report <since>
	<project> - project name
	<since> - YYYY-MM-DD (default to start of today)
`
)

var (
	unknownProject = "<unknown>"
)

type config struct {
	APIToken  string `json:"api_token"`
	Workspace string `json:"workspace"`
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

func loadConfig(cfg *config) error {
	fname, err := configFile()
	if err != nil {
		return err
	}

	file, err := os.Open(fname) // #nosec
	if err != nil {
		return err
	}
	defer file.Close() // #nosec

	return json.NewDecoder(file).Decode(cfg)
}

func printProjects(c *client.Client, prjs []client.Project) {
	names := make([]string, 0, len(prjs))
	clients, err := c.Clients()
	if err != nil {
		log.Printf("warning: can't get clients - %s", err)
	}
	for _, prj := range prjs {
		name := prj.Name
		client := clients[prj.ClientID]
		if client != "" {
			name = fmt.Sprintf("%s/%s", client, name)
		}
		names = append(names, name)
	}

	cmp := func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	}

	sort.Slice(names, cmp)
	for _, name := range names {
		fmt.Println(name)
	}
}

func findProject(name string, prjs []client.Project) []client.Project {
	var matches []client.Project
	name = strings.ToLower(name)
	for _, prj := range prjs {
		if strings.HasPrefix(strings.ToLower(prj.Name), name) {
			matches = append(matches, prj)
		}
	}
	return matches
}

func nameFromID(id int, prjs []client.Project) string {
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

	var cfg config
	if err := loadConfig(&cfg); err != nil {
		log.Fatalf("error: can't load config - %s", err)
	}

	c, err := client.New(cfg.APIToken, cfg.Workspace)
	if err != nil {
		log.Fatalf("error: can't create client: %s", err)
	}

	prjs, err := c.Projects()
	if err != nil {
		log.Fatalf("error: can't get projects - %s", err)
	}

	curTimer, err := c.Timer()
	if err != nil {
		log.Fatalf("error: can't get current timer - %s", err)
	}

	switch command {
	case "projects":
		printProjects(c, prjs)
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
		if err := c.Start(matches[0].ID); err != nil {
			log.Fatalf("error: can't start timer - %s", err)
		}
	case "stop":
		if curTimer == nil {
			log.Fatalf("error: no timer running")
			return // make linter happy
		}
		pid, dur, err := c.Stop(curTimer.ID)
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

		reps, err := c.Report(since)
		if err != nil {
			log.Fatalf("error: can't get report: %s", err)
		}

		for _, r := range reps {
			fmt.Printf("%s: %s\n", r.Project, r.Duration)
		}
	}
}
