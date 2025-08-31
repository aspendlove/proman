package database

import (
	"fmt"
	"os"
	"proman/config"
)

// Diff compares two project schemas and prints the resulting migration script to stdout.
func Diff(cfg *config.Config, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("diff command requires exactly two project IDs (source and target)")
	}
	sourceID := args[0]
	targetID := args[1]

	// 1. Pre-flight Checks
	sourceParams, found := cfg.GetConnection(sourceID)
	if !found {
		return fmt.Errorf("source project with ID '%s' not found", sourceID)
	}
	targetParams, found := cfg.GetConnection(targetID)
	if !found {
		return fmt.Errorf("target project with ID '%s' not found", targetID)
	}

	binaries := cfg.GetBinaryPaths()

	// 2. Generate Diff
	fmt.Fprintf(os.Stderr, "--- Generating Schema Diff for %s -> %s ---", sourceID, targetID)
	migrationScript, err := generateDiff(sourceParams, targetParams, binaries)
	if err != nil {
		return err
	}

	if len(migrationScript) == 0 {
		fmt.Fprintf(os.Stderr, "Schemas are already identical.\n")
		return nil
	}

	// 3. Print Script to stdout
	fmt.Print(migrationScript)

	return nil
}
