package main

import (
	"cmp"
	"encoding/json"
	"errors"
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

	"runtime/debug"

	"charm.land/huh/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"

	"github.com/tebeka/toggl/client"
)

const (
	rcEnvKey = "TOGGLRC"
)

var unknownProject = "<unknown>"

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
		APIToken  string `json:"api_token"` //#nosec
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

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

func selectProject(prjs []client.Project) (client.Project, error) {
	if len(prjs) == 0 {
		return client.Project{}, fmt.Errorf("no projects found")
	}

	slices.SortFunc(prjs, func(p1, p2 client.Project) int {
		return cmp.Compare(strings.ToLower(p1.FullName()), strings.ToLower(p2.FullName()))
	})

	options := make([]huh.Option[int], len(prjs))
	selected := prjs[0].ID
	for i, prj := range prjs {
		options[i] = huh.NewOption(prj.FullName(), prj.ID)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Project").
				Description("Select a project to start").
				Filtering(true).
				Options(options...).
				Value(&selected),
		),
	).WithShowHelp(false).
		WithInput(os.Stdin).
		WithOutput(os.Stderr)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return client.Project{}, fmt.Errorf("project selection canceled")
		}

		return client.Project{}, err
	}

	for _, prj := range prjs {
		if prj.ID == selected {
			return prj, nil
		}
	}

	return client.Project{}, fmt.Errorf("selected unknown project")
}

func resolveStartProject(args []string, prjs []client.Project, interactive bool, chooser func([]client.Project) (client.Project, error)) (client.Project, error) {
	switch len(args) {
	case 0:
		if !interactive {
			return client.Project{}, fmt.Errorf("wrong number of arguments")
		}

		return chooser(prjs)
	case 1:
	default:
		return client.Project{}, fmt.Errorf("wrong number of arguments")
	}

	name := args[0]
	matches := findProject(name, prjs)
	switch len(matches) {
	case 0:
		return client.Project{}, fmt.Errorf("error: no project match %s", name)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, p := range matches {
			names[i] = p.Name
		}

		return client.Project{}, fmt.Errorf("too many matches to %q: %s", name, projectsStr(names))
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

func workspacesCmd(args []string) error {
	fs := flag.NewFlagSet("workspaces", flag.ExitOnError)
	simpleHelp(fs, "workspaces", "List workspaces.")
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

	ws, err := c.Workspaces()
	if err != nil {
		return err
	}

	slices.SortFunc(ws, func(w1, w2 client.Workspace) int {
		return cmp.Compare(w1.Name, w2.Name)
	})

	size := 0
	for _, w := range ws {
		size = max(size, len(w.Name))
	}

	for _, w := range ws {
		fmt.Printf("%-*s %d\n", size, w.Name, w.ID)
	}

	return nil
}

func startCmd(args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	startTime := fs.String("time", "", "start time (HH:MM)")
	simpleHelp(fs, "start [flags] [project]", "Start timer.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 1 || (fs.NArg() == 0 && !isTTY()) {
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

	prj, err := resolveStartProject(fs.Args(), prjs, isTTY(), selectProject)
	if err != nil {
		return err
	}

	fmt.Printf("Starting %s\n", prj.Name)
	return c.Start(prj.ID, start)
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
		return fmt.Errorf("no timer is running")
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
	simpleHelp(fs, "report [YYYY-MM-DD]", "Print report, default to yesterday.")
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
		if _, err := time.Parse("2006-01-02", since); err != nil {
			return fmt.Errorf("date format should be YYYY-MM-DD (got %q)", since)
		}
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

	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}
	fmt.Printf("%s version %s\n", path.Base(os.Args[0]), version)
	return nil
}

type cmd struct {
	name string
	desc string
	fn   func([]string) error
}

var cmds = []cmd{
	{"projects", "show workspace projects", projectsCmd},
	{"report", "print report", reportCmd},
	{"start", "start timer", startCmd},
	{"status", "timer status", statusCmd},
	{"stop", "stop timer", stopCmd},
	{"version", "show version and exit", versionCmd},
	{"workspaces", "show workspaces", workspacesCmd},
}

func printUsage() {
	progName := path.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", progName) //#nosec
	fmt.Fprintf(os.Stderr, "The commands are:\n")
	for _, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "  %s    %s\n", cmd.name, cmd.desc)
	}
	fmt.Fprintf(os.Stderr, `Use "%s <command> -h" for more information about a command.\n`, progName) //#nosec
}

func findCmd(name string) (cmd, error) {
	commandByName := make(map[string]cmd)
	names := make([]string, len(cmds))
	for i, c := range cmds {
		commandByName[c.name] = c
		names[i] = c.name
	}

	matches := fuzzy.Find(name, names)
	switch len(matches) {
	case 0:
		return cmd{}, fmt.Errorf("no command match %q", name)
	case 1:
		return commandByName[matches[0]], nil
	default:
		return cmd{}, fmt.Errorf("too many matches to %q: %s", name, projectsStr(matches))
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmdName := os.Args[1]
	switch cmdName {
	case "-h", "--help":
		printUsage()
		os.Exit(0)
	}

	args := os.Args[2:]

	cmd, err := findCmd(cmdName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		printUsage()
		os.Exit(1)
	}

	if err := cmd.fn(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
