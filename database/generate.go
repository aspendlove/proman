package database

import (
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"proman/utils"
)

func GenTypes(cfg *config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("gen-types command expects exactly one argument: the project ID")
	}
	projectID := args[0]

	params, found := cfg.GetConnection(projectID)
	if !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}

	if params.SupabaseProjectID == "" {
		return fmt.Errorf(
			"supabase project ID is not set for project '%s'. Please add it via the register command or by editing the config file",
			projectID,
		)
	}

	supabasePath := cfg.Binaries.Supabase
	if supabasePath == "" {
		return fmt.Errorf("path to supabase binary is not configured. Please run 'proman init'")
	}

	spin := utils.NewSpinner("Generating TypeScript types for project: %s", projectID)
	spin.Start()
	defer spin.Stop()

	cmd := exec.Command(
		supabasePath,
		"gen",
		"types",
		"--lang",
		"typescript",
		"--project-id",
		params.SupabaseProjectID,
		"--schema",
		"public",
	)

	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run supabase gen types: %w", err)
	}
	return nil
}
