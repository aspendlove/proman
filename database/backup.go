package database

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"strings"
	"time"
)

// backupRoles executes the roles backup safely, excluding a predefined list of internal Supabase roles.
func backupRoles(params config.ConnectionParams, binaries config.BinaryPaths, filename string) error {
	fmt.Printf("Dumping roles to %s...\n", filename)

	// 1. Define the static list of roles to exclude.
	excludedRoles := []string{
		"postgres",
		"anon",
		"authenticated",
		"authenticator",
		"service_role",
		"supabase_admin",
		"supabase_auth_admin",
		"supabase_functions_admin",
		"supabase_read_only_user",
		"supabase_realtime_admin",
		"supabase_replication_admin",
		"supabase_storage_admin",
		"dashboard_user",
		"pgbouncer",
		"pgsodium_keyholder",
		"pgsodium_keyiduser",
		"pgsodium_keymaker",
	}
	excludePattern := strings.Join(excludedRoles, "|")

	// 2. Dump all roles
	dumpCmd := exec.Command(binaries.PGDumpAll, "--roles-only", "--no-role-passwords", "-h", params.Host, "-p", params.Port, "-U", params.User)
	dumpCmd.Env = append(os.Environ(), "PGPASSWORD="+params.Password)

	dumpOutput, err := dumpCmd.Output()
	if err != nil {
		return fmt.Errorf("pg_dumpall command failed: %w", err)
	}

	// 3. Grep to filter out the system roles
	grepCmd := exec.Command("grep", "-vE", "CREATE ROLE ("+excludePattern+");")
	grepCmd.Stdin = bytes.NewReader(dumpOutput)

	filteredOutput, err := grepCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(exitErr.Stderr) > 0 {
				return fmt.Errorf("grep command failed: %s", string(exitErr.Stderr))
			}
		}
	}

	// 4. Write the final output to the file
	return os.WriteFile(filename, filteredOutput, 0644)
}

// backupSchema executes the schema backup safely.
func backupSchema(params config.ConnectionParams, binaries config.BinaryPaths, filename string) error {
	fmt.Printf("Dumping schema to %s...\n", filename)

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	excludedSchemas := []string{"auth", "cron", "extensions", "graphql", "graphql_public", "net", "pgbouncer", "pgsodium", "pgsodium_masks", "realtime", "storage", "supabase_functions", "supabase_migrations", "vault", "_realtime"}

	args := []string{
		"-h", params.Host,
		"-p", params.Port,
		"-U", params.User,
		"-d", params.DBName,
		"--schema-only",
		"--no-owner",
		"--no-privileges",
	}
	for _, s := range excludedSchemas {
		args = append(args, "--exclude-schema="+s)
	}

	cmd := exec.Command(binaries.PGDump, args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+params.Password)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr // Show pg_dump errors to the user

	return cmd.Run()
}

// backupData executes the data backup safely.
func backupData(params config.ConnectionParams, binaries config.BinaryPaths, filename string) error {
	fmt.Printf("Dumping data to %s...\n", filename)

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	excludedSchemas := []string{"auth", "cron", "extensions", "graphql", "graphql_public", "net", "pgbouncer", "pgsodium", "pgsodium_masks", "realtime", "storage", "supabase_functions", "supabase_migrations", "vault", "_realtime"}

	args := []string{
		"-h", params.Host,
		"-p", params.Port,
		"-U", params.User,
		"-d", params.DBName,
		"--data-only",
		"--quote-all-identifiers",
	}
	for _, s := range excludedSchemas {
		args = append(args, "--exclude-schema="+s)
	}

	cmd := exec.Command(binaries.PGDump, args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+params.Password)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr // Show pg_dump errors to the user

	return cmd.Run()
}

// Backup is the main entry point for the backup command.
// It can be called from the CLI or from other functions like Clone.
func Backup(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("backup command requires at least a project ID")
	}

	var projectID string
	var filePrefix string
	doRoles, doSchema, doData := false, false, false

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--roles":
			doRoles = true
		case arg == "--schema":
			doSchema = true
		case arg == "--data":
			doData = true
		case arg == "--prefix":
			if i+1 < len(args) {
				filePrefix = args[i+1]
				i++ // consume the next argument
			} else {
				return fmt.Errorf("--prefix flag requires a value")
			}
		case !strings.HasPrefix(arg, "--"):
			projectID = arg
		default:
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}

	if projectID == "" {
		return fmt.Errorf("no project ID specified")
	}

	// If no specific backup type is requested, do a full backup.
	isFullBackup := !doRoles && !doSchema && !doData
	if isFullBackup {
		doRoles, doSchema, doData = true, true, true
	}

	// If no prefix is provided, create a default one.
	if filePrefix == "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filePrefix = fmt.Sprintf("%s_%s", projectID, timestamp)
	}

	// Get connection params and binary paths
	params, found := cfg.GetConnection(projectID)
	if !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}
	binaries := cfg.GetBinaryPaths()
	if binaries.PSQL == "" || binaries.PGDump == "" || binaries.PGDumpAll == "" {
		return fmt.Errorf("one or more PostgreSQL binary paths are not set in the config")
	}

	// Execute the requested backups
	if doRoles {
		if err := backupRoles(params, binaries, filePrefix+"_roles.sql"); err != nil {
			return fmt.Errorf("failed to backup roles: %w", err)
		}
	}
	if doSchema {
		if err := backupSchema(params, binaries, filePrefix+"_schema.sql"); err != nil {
			return fmt.Errorf("failed to backup schema: %w", err)
		}
	}
	if doData {
		if err := backupData(params, binaries, filePrefix+"_data.sql"); err != nil {
			return fmt.Errorf("failed to backup data: %w", err)
		}
	}

	fmt.Println("Backup complete.")
	return nil
}
