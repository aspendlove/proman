package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"proman/config"
	"proman/database"
	"proman/projects"
)

const helpMessage = `A CLI tool to help manage Supabase projects.

Usage:
  proman <command> <subcommand> [arguments]

COMMANDS:
  connection            Manage project connections.
    register            - Register a new project connection interactively.
    list                - List all registered projects.
    remove [id]         - Remove a registered project.

  db                    Perform database operations.
    backup [id] ...     - Backup a project's database. See 'db backup --help' for flags.
    clone ...           - Safely migrate schema. See 'db clone --help' for flags.
    diff [src] [target] - Show the schema difference between two projects.
    gen-types [id]      - Generate TypeScript types for a project's database.

  supabase              Interact with the Supabase CLI.
    login               - Log in to the Supabase CLI.

  init                  Configure paths to required binaries (psql, results, etc.).
  help                  Show this help message.
`

func main() {
	// --- Load Config ---
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
		log.Println("Configuration file not found. Creating a new one at:", configFile)
		defaultCfg, _ := config.Load(configFile)
		if err := defaultCfg.Save(configFile); err != nil {
			log.Fatalf("Failed to create initial config file: %v", err)
		}
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// --- Command Routing ---
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println(helpMessage)
		return
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "init":
		err = projects.Init(cfg, configFile)
	case "help":
		fmt.Println(helpMessage)
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
			log.Fatal("Error: 'db' requires a subcommand (backup, clone, diff, gen-types).")
		}
		subcommand := commandArgs[0]
		subcommandArgs := commandArgs[1:]
		switch subcommand {
		case "backup":
			err = database.Backup(cfg, subcommandArgs)
		case "clone":
			err = database.Clone(cfg, subcommandArgs)
		case "diff":
			err = database.Diff(cfg, subcommandArgs)
		case "gen-types":
			err = database.GenTypes(cfg, subcommandArgs)
		default:
			log.Fatalf("Error: Unknown subcommand '%s' for 'db'.", subcommand)
		}
	case "supabase":
		if len(commandArgs) < 1 {
			log.Fatal("Error: 'supabase' requires a subcommand (login).")
		}
		subcommand := commandArgs[0]
		subcommandArgs := commandArgs[1:]
		switch subcommand {
		case "login":
			err = projects.Login(cfg, subcommandArgs)
		default:
			log.Fatalf("Error: Unknown subcommand '%s' for 'supabase'.", subcommand)
		}
	default:
		log.Fatalf("Error: Unknown command '%s'.\n\n%s", command, helpMessage)
	}

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
