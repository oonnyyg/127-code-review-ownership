package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	configPath := flag.String("config", "", "path to ownership config JSON (required)")
	changesPath := flag.String("changes", "", "path to change manifest JSON (required)")
	approvalsPath := flag.String("approvals", "", "path to approvals JSON (optional)")
	flag.Parse()

	if *configPath == "" || *changesPath == "" {
		fmt.Fprintln(os.Stderr, "usage: reviewassign -config config.json -changes changes.json [-approvals approvals.json]")
		os.Exit(2)
	}
	if err := run(*configPath, *changesPath, *approvalsPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(configPath, changesPath, approvalsPath string) error {
	var cfg Config
	if err := readJSON(configPath, &cfg); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	var cs ChangeSet
	if err := readJSON(changesPath, &cs); err != nil {
		return fmt.Errorf("changes: %w", err)
	}
	approvals := []string{}
	if approvalsPath != "" {
		var a Approvals
		if err := readJSON(approvalsPath, &a); err != nil {
			return fmt.Errorf("approvals: %w", err)
		}
		approvals = a.Approvals
	}
	res, err := NewResolver(&cfg)
	if err != nil {
		return err
	}
	rep, err := buildReport(res, &cs, approvals)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
