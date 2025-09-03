package database

import (
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"proman/utils"
)

func Exec(cfg *config.Config, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("exec command expects exactly two arguments: the project ID and the filename")
	}

	projectID := args[0]
	filename := args[1]

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filename)
	}

	params, found := cfg.GetConnection(projectID)
	if !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}

	binaries := cfg.GetBinaryPaths()
	if binaries.PSQL == "" {
		return fmt.Errorf("path to psql binary is not set in the config. Please run 'proman init'")
	}

	spin := utils.NewSpinner("Executing SQL File")
	spin.Start()
	defer spin.Stop()

	cmd := exec.Command(binaries.PSQL, "-h", params.Host, "-p", params.Port, "-U", params.User, "-d", params.DBName, "-f", filename)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+params.Password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute psql: %w", err)
	}

	utils.SuccessPrint("Execution complete\n")
	return nil
}
