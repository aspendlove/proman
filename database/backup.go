package database

import (
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"strings"
	"time"
)

func backupRoles(params config.ConnectionParams, binaries config.BinaryPaths, filename string) error {
	fmt.Printf("Dumping roles to %s...\n", filename)

	dumpCmd := exec.Command(
		binaries.PGDumpAll, "--roles-only", "--no-role-passwords", "-h", params.Host, "-p", params.Port, "-U", params.User,
	)
	dumpCmd.Env = append(os.Environ(), "PGPASSWORD="+params.Password)

	dumpOutput, err := dumpCmd.Output()
	if err != nil {
		return fmt.Errorf("pg_dumpall command failed: %w", err)
	}

	return os.WriteFile(filename, dumpOutput, 0644)
}

func backupSchema(params config.ConnectionParams, binaries config.BinaryPaths, filename string) error {
	fmt.Printf("Dumping schema to %s...\n", filename)

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	excludedSchemas := []string{
		"auth", "cron", "extensions", "graphql", "graphql_public", "net", "pgbouncer", "pgsodium", "pgsodium_masks",
		"realtime", "storage", "supabase_functions", "supabase_migrations", "vault", "_realtime",
	}

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
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func backupData(params config.ConnectionParams, binaries config.BinaryPaths, filename string) error {
	fmt.Printf("Dumping data to %s...\n", filename)

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	excludedSchemas := []string{
		"auth", "cron", "extensions", "graphql", "graphql_public", "net", "pgbouncer", "pgsodium", "pgsodium_masks",
		"realtime", "storage", "supabase_functions", "supabase_migrations", "vault", "_realtime",
	}

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
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func Backup(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("backup command requires at least a project ID")
	}

	var projectID string
	var filePrefix string
	doRoles, doSchema, doData := false, false, false

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
				i++
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

	isFullBackup := !doRoles && !doSchema && !doData
	if isFullBackup {
		doRoles, doSchema, doData = true, true, true
	}

	if filePrefix == "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filePrefix = fmt.Sprintf("%s_%s", projectID, timestamp)
	}

	params, found := cfg.GetConnection(projectID)
	if !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}
	binaries := cfg.GetBinaryPaths()
	if binaries.PSQL == "" || binaries.PGDump == "" || binaries.PGDumpAll == "" {
		return fmt.Errorf("one or more PostgreSQL binary paths are not set in the config")
	}

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
