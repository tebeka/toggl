package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	"github.com/tebeka/toggl/togglv8"
	"github.com/urfave/cli"
)

const (
	// Version is current version
	Version  = "0.1.2"
	rcEnvKey = "TOGGLRC"
)

var (
	config struct {
		APIToken  string `json:"api_token"`
		Workspace string `json:"workspace"`
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "toggl"
	app.Version = Version
	app.Usage = "Time tracking app CLI."

	if err := loadConfig(); err != nil {
		log.Fatalf("error: can't load config - %s", err)
	}

	tc := togglv8.New(config.APIToken, config.Workspace)

	// Support for parsing natural language of time.
	w := when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "Start timer",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "description",
					Usage: "Description for the entry",
				},
				cli.StringFlag{
					Name:  "at",
					Value: "now",
					Usage: "When to start the timer",
				},
				cli.StringFlag{
					Name:  "stop",
					Usage: "When to stop the timer",
				},
			},
			Action: func(c *cli.Context) error {
				curTimer, err := tc.CurrentTimer()
				if err != nil {
					return err
				}

				if curTimer != nil {
					return fmt.Errorf("error: there is a timer running")
				}

				prjs, err := tc.Projects()
				if err != nil {
					return err
				}

				if c.NArg() != 1 {
					return fmt.Errorf("wrong number of arguments")
				}

				name := c.Args().First()
				ids := findProject(name, prjs)

				startTime, err := w.Parse(c.String("at"), time.Now())
				if err != nil || startTime == nil {
					return fmt.Errorf("error: failed to parse flag '--at'")
				}

				st, err := w.Parse(c.String("stop"), time.Now())
				if err != nil {
					return fmt.Errorf("error: failed to parse flag '--stop'")
				}

				var stopTime *time.Time
				if st != nil {
					stopTime = &st.Time
				}

				switch len(ids) {
				case 0:
					return fmt.Errorf("error: no project match %s", name)
				case 1:
				default:
					return fmt.Errorf("error: too project many matches to %s", name)
				}

				te, err := tc.StartTimer(ids[0], startTime.Time, stopTime, c.String("description"))
				if err != nil {
					return fmt.Errorf("error: can't start timer - %s", err)
				}

				fmt.Printf("started timer at %s\n", te.Start.Format(time.Stamp))
				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "Stop timer",
			Action: func(c *cli.Context) error {
				curTimer, err := tc.CurrentTimer()
				if err != nil {
					return err
				}

				if curTimer == nil {
					return fmt.Errorf("error: no timer running")
				}

				dur, err := tc.StopTimer(curTimer.ID)
				if err != nil {
					return fmt.Errorf("error: can't stop timer - %s", err)
				}

				fmt.Printf("%s\n", duration2str(dur))

				return nil
			},
		},
		{
			Name:  "status",
			Usage: "Status of current timer",
			Action: func(c *cli.Context) error {
				curTimer, err := tc.CurrentTimer()
				if err != nil {
					return err
				}

				if curTimer == nil {
					return fmt.Errorf("error: no timer running")
				}

				dur := time.Since(curTimer.Start)
				fmt.Printf("duration: %s\n", duration2str(dur))

				return nil
			},
		},
		{
			Name:  "projects",
			Usage: "List projects",
			Action: func(c *cli.Context) error {
				prjs, err := tc.Projects()
				if err != nil {
					return err
				}

				var names []string
				for _, prj := range prjs {
					names = append(names, prj.Name)
				}

				sort.Strings(names)
				for _, name := range names {
					fmt.Printf("* %s\n", name)
				}

				return nil
			},
		},
		{
			Name:  "entries",
			Usage: "Time entries for the past 9 days",
			Action: func(c *cli.Context) error {
				prjs, err := tc.Projects()
				if err != nil {
					return err
				}

				// Map the projects up to be able to display
				// the names.
				projects := make(map[int]string, 0)
				for _, p := range prjs {
					projects[p.ID] = p.Name
				}

				entries, err := tc.Timers()
				if err != nil {
					return err
				}

				for _, entry := range entries {
					var ongoing bool

					duration := time.Duration(time.Duration(entry.Duration) * time.Second)
					if duration < 0 {
						ongoing = true
						duration = time.Since(entry.Start)
					}

					fmt.Printf(
						"* %s - %s > %s",
						entry.Start.Format(time.Stamp),
						entry.Stop.Format(time.Stamp),
						duration2str(duration),
					)

					fmt.Printf(" | %s", projects[entry.PID])

					if ongoing {
						fmt.Printf(" [running]")
					}

					if entry.Description != "" {
						fmt.Printf(": %s", entry.Description)
					}

					fmt.Println()
				}

				return nil
			},
		},
		{
			Name:  "week",
			Usage: "Duration of all entries this week",
			Action: func(c *cli.Context) error {
				entries, err := tc.Timers()
				if err != nil {
					return err
				}

				var durationSum time.Duration
				_, thisWeek := time.Now().ISOWeek()

				for _, entry := range entries {
					_, weekStart := entry.Start.ISOWeek()
					// _, weekStop := entry.Start.ISOWeek() XXX only care about started for now

					if weekStart != thisWeek {
						continue
					}

					duration := time.Duration(time.Duration(entry.Duration) * time.Second)
					if duration < 0 {
						duration = time.Since(entry.Start)
					}

					durationSum += duration
				}

				fmt.Println(duration2str(durationSum))

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func findProject(name string, prjs []togglv8.Project) []int {
	var matches []int
	name = strings.ToLower(name)
	for _, prj := range prjs {
		if strings.HasPrefix(strings.ToLower(prj.Name), name) {
			matches = append(matches, prj.ID)
		}
	}
	return matches
}

func duration2str(dur time.Duration) string {
	h, m, s := int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
