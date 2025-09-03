package database

import (
	"fmt"
	"proman/config"
	"proman/utils"
)

func Diff(cfg *config.Config, args []string) error {
	var err error = nil
	if len(args) != 2 {
		return fmt.Errorf("diff command requires exactly two project IDs (source and target)")
	}
	sourceID := args[0]
	targetID := args[1]

	sourceParams, found := cfg.GetConnection(sourceID)
	if !found {
		return fmt.Errorf("source project with ID '%s' not found", sourceID)
	}
	targetParams, found := cfg.GetConnection(targetID)
	if !found {
		return fmt.Errorf("target project with ID '%s' not found", targetID)
	}

	sourceSchema, err := backupSchema(sourceParams, cfg.Binaries, "")
	if err != nil {
		return fmt.Errorf("could not backup project %s: %w", sourceID, err)
	}
	targetSchema, err := backupSchema(targetParams, cfg.Binaries, "")
	if err != nil {
		return fmt.Errorf("could not backup project %s: %w", sourceID, err)
	}

	return utils.OpenDiff(targetSchema, sourceSchema, "Target", "Source", cfg)
}
