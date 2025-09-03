package database

import (
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"strings"
	"time"
)

func backupRoles(params config.ConnectionParams, binaries config.BinaryPaths, filename string) (string, error) {
	var outFile *os.File = nil
	var err error = nil
	isTempFile := filename == ""

	if isTempFile {
		outFile, err = os.CreateTemp("", "tmp_roles_*.sql")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary output file: %w", err)
		}
		filename = outFile.Name()
	} else {
		outFile, err = os.Create(filename)
		if err != nil {
			return "", fmt.Errorf("failed to create output file: %w", err)
		}
	}
	defer outFile.Close()

	fmt.Printf("Dumping roles to %s...\n", filename)

	cmd := exec.Command(
		binaries.PGDumpAll, "--roles-only", "--no-role-passwords", "-h", params.Host, "-p", params.Port, "-U", params.User,
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+params.Password)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		if isTempFile {
			os.Remove(filename)
		}
		return "", err
	}

	return filename, nil
}

func backupSchema(params config.ConnectionParams, binaries config.BinaryPaths, filename string) (string, error) {
	var outFile *os.File = nil
	var err error = nil
	isTempFile := filename == ""

	if isTempFile {
		outFile, err = os.CreateTemp("", "tmp_schema_*.sql")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary output file: %w", err)
		}
		filename = outFile.Name()
	} else {
		outFile, err = os.Create(filename)
		if err != nil {
			return "", fmt.Errorf("failed to create output file: %w", err)
		}
	}
	defer outFile.Close()

	fmt.Printf("Dumping schema to %s...\n", filename)

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

	err = cmd.Run()

	if err != nil {
		if isTempFile {
			os.Remove(filename)
		}
		return "", err
	}

	return filename, nil
}

func backupData(params config.ConnectionParams, binaries config.BinaryPaths, filename string) (string, error) {
	var outFile *os.File = nil
	var err error = nil
	isTempFile := filename == ""

	if isTempFile {
		outFile, err = os.CreateTemp("", "tmp_data_*.sql")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary output file: %w", err)
		}
		filename = outFile.Name()
	} else {
		outFile, err = os.Create(filename)
		if err != nil {
			return "", fmt.Errorf("failed to create output file: %w", err)
		}
	}
	defer outFile.Close()

	fmt.Printf("Dumping data to %s...\n", filename)

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

	err = cmd.Run()
	if err != nil {
		if isTempFile {
			os.Remove(filename)
		}
		return "", err
	}

	return filename, nil
}

type OfficialType int

const (
	DATA_ONLY OfficialType = iota
	SCHEMA_ONLY
	ROLES_ONLY
)

func officialBackup(params config.ConnectionParams, binaries config.BinaryPaths, filename string, dumpType OfficialType) error {
	connection := FormatRemoteConnectionString(params)

	args := []string{
		"db",
		"dump",
		"--db-url",
		connection,
		"--file",
		filename,
	}

	switch dumpType {
	case DATA_ONLY:
		args = append(args, "--data-only")
	case ROLES_ONLY:
		args = append(args, "--role-only")
	}

	cmd := exec.Command(
		binaries.Supabase,
		args...,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	return cmd.Run()
}

func Backup(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("backup command requires at least a project ID")
	}

	var projectID string
	var filePrefix string
	doRoles, doSchema, doData, doOfficial := false, false, false, false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--official":
			doOfficial = true
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

	isFullBackup := !doRoles && !doSchema && !doData
	if isFullBackup {
		doRoles, doSchema, doData = true, true, true
	}

	if doOfficial {
		// make sure the supabase docker containers get cloes after every backup finishes
		defer func() {
			stopCmd := exec.Command(binaries.Supabase, "stop")
			stopCmd.Run()
		}()
	}

	if doRoles {
		if doOfficial {
			if err := officialBackup(params, binaries, filePrefix+"_roles_official.sql", ROLES_ONLY); err != nil {
				return fmt.Errorf("failed to backup roles: %w", err)
			}
		} else {
			if _, err := backupRoles(params, binaries, filePrefix+"_roles.sql"); err != nil {
				return fmt.Errorf("failed to backup roles: %w", err)
			}
		}
	}
	if doSchema {
		if doOfficial {
			if err := officialBackup(params, binaries, (filePrefix + "_schema_official.sql"), SCHEMA_ONLY); err != nil {
				return fmt.Errorf("failed to backup schema: %w", err)
			}
		} else {
			if _, err := backupSchema(params, binaries, (filePrefix + "_schema.sql")); err != nil {
				return fmt.Errorf("failed to backup schema: %w", err)
			}
		}
	}
	if doData {
		if doOfficial {
			if err := officialBackup(params, binaries, filePrefix+"_data_official.sql", DATA_ONLY); err != nil {
				return fmt.Errorf("failed to backup data: %w", err)
			}
		} else {
			if _, err := backupData(params, binaries, filePrefix+"_data.sql"); err != nil {
				return fmt.Errorf("failed to backup data: %w", err)
			}
		}
	}

	fmt.Println("Backup complete.")
	return nil
}
