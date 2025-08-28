package database

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"proman/config"
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

	fmt.Fprintf(os.Stderr, "Generating TypeScript types for project: %s (%s)\n", projectID, params.SupabaseProjectID)

	cmd := exec.Command(
		supabasePath, "gen", "types", "--lang", "typescript", "--project-id", params.SupabaseProjectID, "--schema",
		"public",
	)

	output, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("failed to run supabase gen types: %s", string(ee.Stderr))
		}
		return fmt.Errorf("failed to run supabase gen types: %w", err)
	}

	fmt.Println(string(output))

	return nil
}
