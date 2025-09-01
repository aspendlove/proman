package database

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"proman/config"
	"strings"
	"time"

	_ "github.com/lib/pq"
	pgschemadiff "github.com/stripe/pg-schema-diff/pkg/diff"
)

func formatRemoteConnectionString(connection config.ConnectionParams) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		connection.User, connection.Password, connection.Host, connection.Port, connection.DBName,
	)
}

func generateDiffStripe(sourceParams, targetParams config.ConnectionParams, binaries config.BinaryPaths) (string, error) {
	sourceDSN := formatRemoteConnectionString(sourceParams)
	targetDSN := formatRemoteConnectionString(sourceParams)

	// Open database connections
	sourceDB, err := sql.Open("postgres", sourceDSN)
	if err != nil {
		return "", fmt.Errorf("failed to open source database connection: %w", err)
	}
	defer sourceDB.Close()

	targetDB, err := sql.Open("postgres", targetDSN)
	if err != nil {
		return "", fmt.Errorf("failed to open target database connection: %w", err)
	}
	defer targetDB.Close()

	// Set connection timeouts (e.g., 10 seconds)
	sourceDB.SetConnMaxLifetime(10 * time.Second)
	sourceDB.SetMaxOpenConns(1)
	targetDB.SetConnMaxLifetime(10 * time.Second)
	targetDB.SetMaxOpenConns(1)

	// Ping databases to ensure connections are established
	if err := sourceDB.Ping(); err != nil {
		return "", fmt.Errorf("failed to connect to source database: %w", err)
	}
	if err := targetDB.Ping(); err != nil {
		return "", fmt.Errorf("failed to connect to target database: %w", err)
	}

	// Generate the diff
	plan, err := pgschemadiff.Generate(
		context.Background(), pgschemadiff.DBSchemaSource(sourceDB), pgschemadiff.DBSchemaSource(targetDB),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema diff: %w", err)
	}

	// Format the diff into a single SQL script
	var migrationScript strings.Builder
	for _, stmt := range plan.Statements {
		migrationScript.WriteString(stmt.ToSQL())
		migrationScript.WriteString(";\n") // Add semicolon and newline for readability
	}

	return migrationScript.String(), nil
}

func generateDiffSupabase(sourceParams, targetParams config.ConnectionParams, binaries config.BinaryPaths) (string, error) {
	tempDir, err := os.MkdirTemp("", "supabase_project-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sourceUrl := formatRemoteConnectionString(sourceParams)
	targetUrl := formatRemoteConnectionString(targetParams)

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

// Clone performs a safe schema migration from a source to a target database.
func Clone(cfg *config.Config, args []string) error {
	var sourceID, targetID string

	// 1. Parse arguments
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

	// 2. Pre-flight Checks
	fmt.Println("--- Running Pre-flight Checks ---")
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
	fmt.Println("Checks passed.")

	// 3. Automatic Backups
	fmt.Println("\n--- Performing Safety Backups ---")
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	fmt.Printf("Backing up source project '%s'...\n", sourceID)
	sourcePrefix := fmt.Sprintf("%s_clone_backup_%s", sourceID, timestamp)
	if err := Backup(cfg, []string{sourceID, "--prefix", sourcePrefix}); err != nil {
		return fmt.Errorf("failed to backup source project '%s': %w", sourceID, err)
	}

	fmt.Printf("Backing up target project '%s'...", targetID)
	targetPrefix := fmt.Sprintf("%s_clone_backup_%s", targetID, timestamp)
	if err := Backup(cfg, []string{targetID, "--prefix", targetPrefix}); err != nil {
		return fmt.Errorf("failed to backup target project '%s': %w", targetID, err)
	}
	fmt.Println("Backups complete.")

	// 4. Generate Diff
	fmt.Println("\n--- Generating Schema Diff ---")
	migrationScript, err := generateDiffSupabase(sourceParams, targetParams, binaries)
	if err != nil {
		return err
	}

	if len(migrationScript) == 0 {
		fmt.Println("Schemas are already identical. No migration needed.")
		return nil
	}

	fmt.Println("Diff generated successfully.")

	// 5. User Confirmation
	fmt.Println("\n--- Review Migration Script ---")

	// Create a temporary file to hold the script
	tmpfile, err := os.CreateTemp("", "proman_migration_*.sql")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for migration script: %w", err)
	}
	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name()) // Clean up the file afterwards

	if _, err := tmpfile.WriteString(migrationScript); err != nil {
		return fmt.Errorf("failed to write migration script to temporary file: %w", err)
	}

	fmt.Println("Opening migration script in `less` for review (press 'q' to quit)... ")

	// Run less to display the file

	lessCmd := exec.Command("less", tmpfile.Name())

	lessCmd.Stdout = os.Stdout

	lessCmd.Stderr = os.Stderr

	lessCmd.Stdin = os.Stdin // Allow less to be controlled by the user

	if err := lessCmd.Run(); err != nil {
		// An error from `less` is not critical, but we should inform the user.
		// The script can still be viewed in the temp file.
		fmt.Fprintf(os.Stderr, "Warning: could not open script in `less`: %v\n", err)
		fmt.Fprintf(os.Stderr, "The script can be found at: %s\n", tmpfile.Name())
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Are you sure you want to apply this migration to project '%s'? (y/n): ", targetID)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	if strings.TrimSpace(strings.ToLower(response)) != "y" {
		fmt.Println("Migration cancelled by user.")
		return nil
	}

	// 6. Apply Migration
	fmt.Println("\n--- Applying Migration ---")
	// Construct URL with password.
	targetURL := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s", targetParams.User, targetParams.Password, targetParams.Host, targetParams.Port,
		targetParams.DBName,
	)
	applyCmd := exec.Command(binaries.PSQL, "-d", targetURL, "-c", migrationScript)
	// No environment variable needed for psql when password is in the URL.
	applyCmd.Env = os.Environ()
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply migration: %s\n%w", string(applyOutput), err)
	}

	fmt.Println(string(applyOutput))
	fmt.Println("Migration applied successfully.")

	return nil
}
