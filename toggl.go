package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
)

const (
	APIBase = "https://www.toggl.com/api/v8"
)

var (
	baseURL = fmt.Sprintf("%s/time_entries", APIBase)
	config  struct {
		APIToken  string `json:"api_token"`
		Workspace string `json:"workspace"`
	}
)

type Project struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
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

func getProjects() ([]Project, error) {
	url := fmt.Sprintf("%s/workspaces/%s/projects", APIBase, config.Workspace)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(config.APIToken, "api_token")
	resp, err := http.DefaultClient.Do(req)
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

func main() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	prjs, err := getProjects()
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range prjs {
		fmt.Printf("%+v\n", p)
	}
}
