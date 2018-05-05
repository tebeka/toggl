package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
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
	if err := dec.Decode(&config); err != nil {
		return err
	}

	return nil
}
