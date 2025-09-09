package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"proman/config"
	"proman/database"
	"proman/projects"
	"proman/utils"
)

const helpMessage = `A powerful CLI tool designed to streamline the management of Supabase projects and their databases.

Usage:
  proman <command> [subcommand] [arguments]

COMMAND GROUPS:
  connection: Manage project connection configurations.
    proman connection register
        Interactively registers a new project connection by prompting for ID, host, user, etc.

    proman connection list
        Lists all currently registered project connections in a table format.

    proman connection remove [project-id]
        Removes a registered project connection by its unique ID.

  db: Perform powerful database operations like backups, migrations, and diffs.
    proman db backup [project-id] [flags]
        Backs up a project's database. By default, performs a full backup (roles, schema, data).
        Arguments:
          [project-id]      The ID of the project to back up.
        Flags:
          --roles           Backup only the roles.
          --schema          Backup only the database schema.
          --data            Backup only the data.
          --prefix [prefix] Set a custom prefix for the backup filenames.
          --official        Use the official 'supabase' CLI for the backup process.

    proman db exec [project-id] [filename]
        Executes a given .sql file against a specified project's database.
        Arguments:
          [project-id]      The ID of the project to execute the file against.
          [filename]        The path to the .sql file to be executed.

    proman db clone --source [id] --target [id]
        Safely migrates the schema of a target database to match a source database.
        This is a safe operation that backs up both databases, generates a migration script,
        prompts for user review and confirmation, and then applies the migration.
        Flags:
          --source [id]     The project ID to use as the desired schema source.
          --target [id]     The project ID of the database to be migrated.

    proman db diff [source-id] [target-id]
        Generates and displays a schema diff between two projects.
        Arguments:
          [source-id]       The project ID to use as the source of truth.
          [target-id]       The project ID to compare against the source.

    proman db gen-migration [source-id] [target-id]
        Generates a migration SQL script to make the target schema match the source.
        The script is printed to standard output and is NOT automatically applied.
        Arguments:
          [source-id]       The project ID with the desired schema.
          [target-id]       The project ID of the database to be migrated.

    proman db gen-types [project-id]
        Generates TypeScript types for the 'public' schema of a project's database.
        Arguments:
          [project-id]      The ID of the project.

  supabase: Interact directly with the Supabase CLI.
    proman supabase login
        A convenient wrapper for the 'supabase login' command.

TOP-LEVEL COMMANDS:
  proman init
      Interactively configures the paths for required binaries (psql, pg_dump, etc.)
      and sets the preferred diff viewer.

  proman help
      Shows this help message.
`

func main() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Failed to get user config directory: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "proman")
	configFile := filepath.Join(configDir, "config.json")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		utils.InfoPrint("Configuration file not found, creating a new one at: %s", configFile)
		defaultCfg, _ := config.Load(configFile)
		if err := defaultCfg.Save(configFile); err != nil {
			log.Fatalf("Failed to create initial config file: %v", err)
		}
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	args := os.Args[1:]
	if len(args) < 1 {
		utils.PrettyPrint(helpMessage)
		return
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "init":
		err = projects.Init(cfg, configFile)
	case "help":
		utils.PrettyPrint(helpMessage)
	case "connection":
		if len(commandArgs) < 1 {
			log.Fatal("Error: 'connection' requires a subcommand (register, list, remove).")
		}
		subcommand := commandArgs[0]
		subcommandArgs := commandArgs[1:]
		switch subcommand {
		case "register":
			err = projects.Register(cfg, configFile, subcommandArgs)
		case "list":
			err = projects.List(cfg, subcommandArgs)
		case "remove":
			err = projects.Remove(cfg, configFile, subcommandArgs)
		default:
			log.Fatalf("Error: Unknown subcommand '%s' for 'connection'.", subcommand)
		}
	case "db":
		if len(commandArgs) < 1 {
			log.Fatal("Error: 'db' requires a subcommand (backup, clone, diff, gen-types, gen-migration).")
		}
		subcommand := commandArgs[0]
		subcommandArgs := commandArgs[1:]
		switch subcommand {
		case "backup":
			err = database.Backup(cfg, subcommandArgs)
		case "exec":
			err = database.Exec(cfg, subcommandArgs)
		case "clone":
			err = database.Clone(cfg, subcommandArgs)
		case "diff":
			err = database.Diff(cfg, subcommandArgs)
		case "gen-types":
			err = database.GenTypes(cfg, subcommandArgs)
		case "gen-migration":
			err = database.GenMigration(cfg, subcommandArgs)
		default:
			log.Fatalf("Error: Unknown subcommand '%s' for 'db'.", subcommand)
		}
	case "supabase":
		if len(commandArgs) < 1 {
			log.Fatal("Error: 'supabase' requires a subcommand (login).")
		}
		cmd := exec.Command(
			cfg.Binaries.Supabase,
			commandArgs...,
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		// subcommand := commandArgs[0]
		// subcommandArgs := commandArgs[1:]
		// switch subcommand {
		// case "login":
		// 	err = projects.Login(cfg, subcommandArgs)
		// default:
		// 	log.Fatalf("Error: Unknown subcommand '%s' for 'supabase'.", subcommand)
		// }
	default:
		log.Fatalf("Error: Unknown command '%s'.\n\n%s", command, helpMessage)
	}

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
