// Command reviewassign explains code-review ownership offline.
// It reads a JSON ownership config and a JSON change list and prints
// a JSON report; it never contacts GitHub or the filesystem beyond
// the two input files.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	review "example.com/code-review-ownership"
)

func main() {
	configPath := flag.String("config", "", "path to the JSON ownership config (required)")
	changesPath := flag.String("changes", "", "path to the JSON change list (required)")
	flag.Parse()
	if *configPath == "" || *changesPath == "" {
		fmt.Fprintln(os.Stderr, "usage: reviewassign -config config.json -changes changes.json")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if err := run(*configPath, *changesPath); err != nil {
		fmt.Fprintln(os.Stderr, "reviewassign:", err)
		os.Exit(1)
	}
}

func run(configPath, changesPath string) error {
	cfgFile, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer cfgFile.Close()
	cfg, err := review.LoadConfig(cfgFile)
	if err != nil {
		return err
	}

	chgFile, err := os.Open(changesPath)
	if err != nil {
		return err
	}
	defer chgFile.Close()
	cs, err := review.LoadChangeSet(chgFile)
	if err != nil {
		return err
	}

	rep := review.Resolve(cfg, cs)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}
