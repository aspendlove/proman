package database

import (
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

	// Get the connection details from the config
	params, found := cfg.GetConnection(projectID)
	if !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}

	if params.SupabaseProjectID == "" {
		return fmt.Errorf("supabase project ID is not set for project '%s'. Please add it via the register command or by editing the config file", projectID)
	}

	supabasePath := cfg.Binaries.Supabase
	if supabasePath == "" {
		return fmt.Errorf("path to supabase binary is not configured. Please run 'proman init'")
	}

	fmt.Fprintf(os.Stderr, "Generating TypeScript types for project: %s (%s)\n", projectID, params.SupabaseProjectID)

	// Construct the command
	cmd := exec.Command(supabasePath, "gen", "types", "--lang", "typescript", "--project-id", params.SupabaseProjectID, "--schema", "public")

	// Capture the output
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, stderr will be in the err object
		if ee, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("failed to run supabase gen types: %s", string(ee.Stderr))
		}
		return fmt.Errorf("failed to run supabase gen types: %w", err)
	}

	// Print the output to stdout
	fmt.Println(string(output))

	return nil
}
