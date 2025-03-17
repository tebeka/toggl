package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"os/user"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/tebeka/toggl/client"
)

const (
	rcEnvKey = "TOGGLRC"
)

var (
	version        = "0.6.1"
	unknownProject = "<unknown>"
)

func configFile() (string, error) {
	if path := os.Getenv(rcEnvKey); len(path) > 0 {
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
		APIToken  string `json:"api_token"`
		Workspace string `json:"workspace"`
		Timeout   string `json:"timeout"`
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

	if timeout <= 0 {
		return client.Config{}, fmt.Errorf("bad timeout - %v", timeout)
	}

	wid, err := strconv.Atoi(cfg.Workspace)
	if err != nil {
		return client.Config{}, fmt.Errorf("bad workspace ID: %w", err)
	}

	c := client.Config{
		APIToken:    cfg.APIToken,
		WorkspaceID: int(wid),
		Timeout:     timeout,
	}

	if err := c.Validate(); err != nil {
		return client.Config{}, err
	}

	return c, nil
}

func findProject(name string, prjs []client.Project) []client.Project {
	name = strings.ToLower(name)
	projects := make(map[string]client.Project)
	for _, prj := range prjs {
		projects[strings.ToLower(prj.Name)] = prj
	}
	names := slices.Collect(maps.Keys(projects))

	matches := fuzzy.Find(name, names)
	out := make([]client.Project, len(matches))
	for i, m := range matches {
		out[i] = projects[m]
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

func exeName() string {
	return path.Base(os.Args[0])
}

func simpleHelp(fs *flag.FlagSet, cmd, desc string) {
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s\n%s\n\n", exeName(), cmd, desc)
		fs.PrintDefaults()
	}
}

func projectsCmd(args []string) error {
	fs := flag.NewFlagSet("projects", flag.ExitOnError)
	simpleHelp(fs, "projects", "List projects.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("wrong number of arguments")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	prjs, err := c.Projects()
	if err != nil {
		return err
	}

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

func startCmd(args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	startTime := fs.String("time", "", "start time (HH:MM)")
	simpleHelp(fs, "start [flags] <project>", "Start timer.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("wrong number of arguments")
	}

	start := time.Now()
	if *startTime != "" {
		t, err := time.Parse("15:04", *startTime)
		if err != nil {
			return fmt.Errorf("start: bad time (should be HH:MM) - %w", err)
		}
		start = time.Date(start.Year(), start.Month(), start.Day(), t.Hour(), t.Minute(), 0, 0, start.Location())
	}

	start = start.In(time.UTC)

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

	name := fs.Arg(0)
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
	return c.Start(matches[0].ID, start)
}

func stopCmd(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	simpleHelp(fs, "stop", "Stop timer.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
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

func statusCmd(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	simpleHelp(fs, "status", "Show timer status.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("wrong number of arguments")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	t, err := c.Timer()
	if err != nil {
		return err
	}

	if t == nil {
		return fmt.Errorf("no time is running")
	}

	dur := time.Since(t.Start)

	prjs, err := c.Projects()
	if err != nil {
		return err
	}

	name := nameFromID(t.Project, prjs)
	if name == "" {
		name = unknownProject
	}

	fmt.Printf("%s: %s\n", name, duration2str(dur))
	return nil
}

func reportCmd(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	simpleHelp(fs, "report [date]", "Print report.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch fs.NArg() {
	case 0, 1:
		// OK
	default:
		return fmt.Errorf("wrong number of arguments")
	}

	yday := time.Now().Add(-24 * time.Hour)
	since := yday.Format("2006-01-02")
	if fs.NArg() == 1 {
		since = fs.Arg(0)
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	reps, err := c.Report(since)
	if err != nil {
		log.Fatalf("error: can't get report: %s", err)
	}

	for _, r := range reps {
		fmt.Printf("%s: %s\n", r.Project, r.Duration)
	}

	return nil
}

func versionCmd(args []string) error {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	simpleHelp(fs, "version", "Show version and exit.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("wrong number of arguments")
	}

	fmt.Printf("%s version %s\n", path.Base(os.Args[0]), version)
	return nil
}

func printUsage() {
	progName := path.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", progName)
	fmt.Fprintf(os.Stderr, "The commands are:\n")
	fmt.Fprintf(os.Stderr, "  version     show version and exit\n")
	fmt.Fprintf(os.Stderr, "  projects    show workspace projects\n")
	fmt.Fprintf(os.Stderr, "  start       start timer\n")
	fmt.Fprintf(os.Stderr, "  stop        stop timer\n")
	fmt.Fprintf(os.Stderr, "  status      timer status\n")
	fmt.Fprintf(os.Stderr, "  report      print report\n\n")
	fmt.Fprintf(os.Stderr, "Use \"%s <command> -h\" for more information about a command.\n", progName)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "version":
		err = versionCmd(args)
	case "projects":
		err = projectsCmd(args)
	case "start":
		err = startCmd(args)
	case "stop":
		err = stopCmd(args)
	case "status":
		err = statusCmd(args)
	case "report":
		err = reportCmd(args)
	case "-h", "--help":
		printUsage()
		os.Exit(0)
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
