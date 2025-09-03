package database

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"proman/config"
	"proman/utils"
	"strings"
	"time"
)

func generateMigration(sourceParams, targetParams config.ConnectionParams, binaries config.BinaryPaths) (string, error) {
	spin := utils.NewSpinner("Generating migrations")
	spin.Start()
	defer spin.Stop()
	tempDir, err := os.MkdirTemp("", "supabase_project-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sourceUrl := FormatRemoteConnectionString(sourceParams)
	targetUrl := FormatRemoteConnectionString(targetParams)

	supabasePath := binaries.Supabase

	cmd := exec.Command(supabasePath, "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to initialize supabase project: %w", err)
	}

	migrationsDir := filepath.Join(tempDir, "supabase", "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create migrations directory: %w", err)
	}

	dumpFilePath := filepath.Join(migrationsDir, "0001_source_schema_dump.sql")
	dumpFile, err := os.Create(dumpFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create dump file: %w", err)
	}
	defer dumpFile.Close()

	cmd = exec.Command(supabasePath, "db", "dump", "--db-url", targetUrl)
	cmd.Stdout = dumpFile
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to dump source schema: %w", err)
	}

	cmd = exec.Command(supabasePath, "start")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to start local supabase instance: %w", err)
	}
	defer func() {
		stopCmd := exec.Command(supabasePath, "stop")
		stopCmd.Dir = tempDir
		stopCmd.Run()
	}()

	cmd = exec.Command(supabasePath, "db", "reset")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to reset local db: %w", err)
	}

	cmd = exec.Command(supabasePath, "db", "diff", "--db-url", sourceUrl)
	cmd.Dir = tempDir
	bytesOut, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create diff against target db: %w", err)
	}

	return string(bytesOut), nil
}

func GenMigration(cfg *config.Config, args []string) error {
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

	binaries := cfg.GetBinaryPaths()
	spin := utils.NewSpinner("Generating diff: %s -> %s", sourceID, targetID)
	spin.Start()
	defer spin.Stop()
	migrationScript, err := generateMigration(sourceParams, targetParams, binaries)
	if err != nil {
		return err
	}

	if len(migrationScript) == 0 {
		utils.WarningPrint("Schemas are already identical\n")
		return nil
	}

	fmt.Print(migrationScript)

	return nil
}

func Clone(cfg *config.Config, args []string) error {
	var sourceID, targetID string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 < len(args) {
				sourceID = args[i+1]
				i++
			} else {
				return fmt.Errorf("--source flag requires a value")
			}
		case "--target":
			if i+1 < len(args) {
				targetID = args[i+1]
				i++
			} else {
				return fmt.Errorf("--target flag requires a value")
			}
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	if sourceID == "" || targetID == "" {
		return fmt.Errorf("both --source and --target flags are required")
	}

	sourceParams, found := cfg.GetConnection(sourceID)
	if !found {
		return fmt.Errorf("source project with ID '%s' not found", sourceID)
	}
	targetParams, found := cfg.GetConnection(targetID)
	if !found {
		return fmt.Errorf("target project with ID '%s' not found", targetID)
	}

	binaries := cfg.GetBinaryPaths()
	if binaries.PSQL == "" || binaries.PGDump == "" || binaries.PGDumpAll == "" {
		return fmt.Errorf("one or more required binaries (psql, pg_dump, pg_dumpall) are not set in the config")
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")

	spin := utils.NewSpinner("Backing up source project '%s'\n", sourceID)
	spin.Start()
	defer func() {
		spin.Stop()
	}()

	sourcePrefix := fmt.Sprintf("%s_clone_backup_%s", sourceID, timestamp)
	if err := Backup(cfg, []string{sourceID, "--prefix", sourcePrefix}); err != nil {
		return fmt.Errorf("failed to backup source project '%s': %w", sourceID, err)
	}

	spin.Stop()
	spin = utils.NewSpinner("Backing up target project '%s'", targetID)
	spin.Start()

	targetPrefix := fmt.Sprintf("%s_clone_backup_%s", targetID, timestamp)
	if err := Backup(cfg, []string{targetID, "--prefix", targetPrefix}); err != nil {
		return fmt.Errorf("failed to backup target project '%s': %w", targetID, err)
	}

	spin.Stop()
	spin = utils.NewSpinner("Generating migrations")
	spin.Start()

	migrationScript, err := generateMigration(sourceParams, targetParams, binaries)
	if err != nil {
		return err
	}

	if len(migrationScript) == 0 {
		utils.WarningPrint("No clone needed\n")
		return nil
	}

	utils.WarningPrint("\n--- Reivew Migration Script ---\n")

	tmpfile, err := os.CreateTemp("", "proman_migration_*.sql")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for migration script: %w", err)
	}
	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(migrationScript); err != nil {
		return fmt.Errorf("failed to write migration script to temporary file: %w", err)
	}

	utils.InfoPrint("Opening migration script in `less` for review (press 'q' to quit)... ")

	lessCmd := exec.Command("less", tmpfile.Name())
	lessCmd.Stdout = os.Stdout
	lessCmd.Stderr = os.Stderr
	lessCmd.Stdin = os.Stdin

	if err := lessCmd.Run(); err != nil {
		utils.WarningPrint("Warning: could not open script in `less`: %w\n", err)
		utils.WarningPrint("The script can be found for review at: %s\n", tmpfile.Name())
	}

	reader := bufio.NewReader(os.Stdin)

	response, err := utils.Prompt(reader, fmt.Sprintf("Are you sure you want to apply this migration to project '%s'? (y/n): ", targetID))
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	if strings.TrimSpace(strings.ToLower(response)) != "y" {
		utils.ErrorPrint("Migration cancelled by user\n")
		return nil
	}

	spin.Stop()
	spin = utils.NewSpinner("Applying migrations")
	spin.Start()

	targetURL := FormatRemoteConnectionString(targetParams)
	applyCmd := exec.Command(binaries.PSQL, "-d", targetURL, "-c", migrationScript)
	applyCmd.Stderr = os.Stderr
	applyCmd.Stdout = os.Stdout
	err = applyCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	utils.SuccessPrint("Migration applied successfully\n")
	return nil
}
