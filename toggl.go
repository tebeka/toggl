package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"

	"github.com/sahilm/fuzzy"
	"github.com/urfave/cli/v2"

	"github.com/tebeka/toggl/client"
)

const (
	rcEnvKey = "TOGGLRC"

	usage = `usage: %s start <project>|stop|status|projects|report <since>
	<project> - project name
	<since>   - YYYY-MM-DD (default to start of today)
`
)

var (
	version        = "0.4.7"
	unknownProject = "<unknown>"
)

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

func loadConfig() (client.Config, error) {
	fname, err := configFile()
	if err != nil {
		return client.Config{}, err
	}

	file, err := os.Open(fname) // #nosec
	if err != nil {
		return client.Config{}, err
	}
	defer file.Close() // #nosec

	var cfg struct {
		APIToken  string
		Workspace int
		Timeout   string
	}

	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return client.Config{}, err
	}

	timeout := 5 * time.Second
	if cfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			return client.Config{}, err
		}
	}

	c := client.Config{
		APIToken:  cfg.APIToken,
		Workspace: cfg.Workspace,
		Timeout:   timeout,
	}
	return c, nil
}

// fuzzy.Source interface
type projects []client.Project

func (ps projects) String(i int) string { return ps[i].FullName() }
func (ps projects) Len() int            { return len(ps) }

func findProject(name string, prjs []client.Project) []client.Project {
	matches := fuzzy.FindFrom(name, projects(prjs))
	out := make([]client.Project, len(matches))
	for i, m := range matches {
		out[i] = prjs[m.Index]
	}
	return out
}

func nameFromID(id int, prjs []client.Project) string {
	for _, prj := range prjs {
		if prj.ID == id {
			return prj.Name
		}
	}

	return ""
}

func duration2str(dur time.Duration) string {
	h, m, s := int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func projectsStr(prjs []string) string {
	s := make([]string, len(prjs))
	copy(s, prjs)
	sort.Strings(s)
	return strings.Join(s, ", ")
}

func newClient() (*client.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return client.New(cfg)
}

func projectsCmd(ctx *cli.Context) error {
	if ctx.NArg() != 0 {
		return fmt.Errorf("wrong number of arguments")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	prjs, err := c.Projects()
	names := make([]string, 0, len(prjs))
	for _, prj := range prjs {
		names = append(names, prj.FullName())
	}

	cmp := func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	}

	sort.Slice(names, cmp)
	for _, name := range names {
		fmt.Println(name)
	}

	return nil
}

func startCmd(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("wrong number of arguments")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	curTimer, err := c.Timer()
	if err != nil {
		return err
	}

	if curTimer != nil {
		return fmt.Errorf("there's a timer running")
	}

	prjs, err := c.Projects()
	if err != nil {
		return err
	}

	name := ctx.Args().Get(0)
	matches := findProject(name, prjs)
	switch len(matches) {
	case 0:
		log.Fatalf("error: no project match %s", name)
	case 1:
	default:
		names := make([]string, len(matches))
		for i, p := range matches {
			names[i] = p.Name
		}

		return fmt.Errorf("too many matches to %q: %s", name, projectsStr(names))
	}

	fmt.Printf("Starting %s\n", matches[0].Name)
	return c.Start(matches[0].ID)
}

func stopCmd(ctx *cli.Context) error {
	if ctx.NArg() != 0 {
		return fmt.Errorf("wrong number of arguments")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	curTimer, err := c.Timer()
	if err != nil {
		return err
	}

	if curTimer == nil {
		return fmt.Errorf("no timer running")
	}

	pid, dur, err := c.Stop(curTimer.ID)
	if err != nil {
		return err
	}

	prjs, err := c.Projects()
	if err != nil {
		return err
	}

	name := nameFromID(pid, prjs)
	if name == "" {
		name = unknownProject
	}
	fmt.Printf("%s: %s\n", name, duration2str(dur))
	return nil
}

func main() {
	log.SetFlags(0) // Don't prefix with time

	app := cli.App{
		Name:  "toggl",
		Usage: "toggl track client",
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "show version and exit",
				Action: func(ctx *cli.Context) error {
					fmt.Printf("%s\n", version)
					return nil
				},
			},
			{
				Name:   "projects",
				Usage:  "show workspace projects",
				Action: projectsCmd,
			},
			{
				Name:   "start",
				Usage:  "start timer",
				Action: startCmd,
			},
			{
				Name:   "stop",
				Usage:  "stop timer",
				Action: stopCmd,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("error: %s", err)
	}

	return

	/*

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
				names := make([]string, len(matches))
				for i, p := range matches {
					names[i] = p.Name
				}

				log.Fatalf("error: too many matches to %q: %s", name, projectsStr(names))
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
	*/
}
