package projects

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"proman/utils"
	"text/tabwriter"

	"github.com/ugurcsen/gods-generic/sets/hashset"
)

func Register(cfg *config.Config, configFile string, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("register command takes no arguments")
	}

	reader := bufio.NewReader(os.Stdin)
	utils.InfoPrint("Registering a new Supabase project connection\n")

	projectID, err := utils.Prompt(reader, "Enter a unique ID for this project: ")
	if err != nil {
		return err
	}
	if _, found := cfg.GetConnection(projectID); found {
		return fmt.Errorf("project with ID '%s' already exists", projectID)
	}

	host, err := utils.Prompt(reader, "Enter Host: ")
	if err != nil {
		return err
	}

	port, err := utils.Prompt(reader, "Enter Port (default: 5432): ")
	if err != nil {
		return err
	}
	if port == "" {
		port = "5432"
	}

	user, err := utils.Prompt(reader, "Enter User: ")
	if err != nil {
		return err
	}

	password, err := utils.Prompt(reader, "Enter Password: ")
	if err != nil {
		return err
	}

	dbName, err := utils.Prompt(reader, "Enter Database Name (default: postgres): ")
	if err != nil {
		return err
	}
	if dbName == "" {
		dbName = "postgres"
	}

	supabaseProjectID, err := utils.Prompt(reader, "Enter Supabase Project ID: ")
	if err != nil {
		return err
	}

	params := config.ConnectionParams{
		Password:          password,
		Host:              host,
		Port:              port,
		User:              user,
		DBName:            dbName,
		SupabaseProjectID: supabaseProjectID,
	}
	cfg.AddConnection(projectID, params)

	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	utils.SuccessPrint("\nSuccessfully registered project %s\n", projectID)
	return nil
}

func List(cfg *config.Config, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("list command takes no arguments")
	}

	connectionIDs := cfg.ListConnections()
	if len(connectionIDs) == 0 {
		utils.WarningPrint("No projects are registered yet. Use 'proman register' to add one")
		return nil
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, ' ', 0)

	fmt.Fprintln(w, "ID\tHOST\tUSER\tDATABASE")
	fmt.Fprintln(w, "--\t----\t----\t--------")

	for _, id := range connectionIDs {
		params, _ := cfg.GetConnection(id)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, params.Host, params.User, params.DBName)
	}

	return w.Flush()
}

func Remove(cfg *config.Config, configFile string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("remove command expects exactly one argument: the project ID")
	}
	projectID := args[0]

	if _, found := cfg.GetConnection(projectID); !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}

	cfg.RemoveConnection(projectID)

	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	utils.SuccessPrint("Successfully removed project %s\n", projectID)
	return nil
}

func Login(cfg *config.Config, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("login command takes no arguments")
	}

	supabasePath := cfg.Binaries.Supabase
	if supabasePath == "" {
		return fmt.Errorf("path to supabase binary is not configured. Please run 'proman init'")
	}

	cmd := exec.Command(supabasePath, "login")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run '%s login': %w", supabasePath, err)
	}

	return nil
}

func Init(cfg *config.Config, configFile string) error {
	reader := bufio.NewReader(os.Stdin)
	utils.InfoPrint("--- Configure Binary Paths ---\n")
	utils.InfoPrint("Please provide the absolute paths for the following tools\n")
	utils.InfoPrint("If a tool is already in your system's PATH, you can just enter its name (ex 'psql')\n")

	psqlPath, err := utils.Prompt(reader, fmt.Sprintf("Path to psql (current: %s): ", cfg.Binaries.PSQL))
	if err != nil {
		return err
	}

	pgDumpPath, err := utils.Prompt(reader, fmt.Sprintf("Path to pg_dump (current: %s): ", cfg.Binaries.PGDump))
	if err != nil {
		return err
	}

	pgDumpAllPath, err := utils.Prompt(reader, fmt.Sprintf("Path to pg_dumpall (current: %s): ", cfg.Binaries.PGDumpAll))
	if err != nil {
		return err
	}

	supabasePath, err := utils.Prompt(reader, fmt.Sprintf("Path to supabase (current: %s): ", cfg.Binaries.Supabase))
	if err != nil {
		return err
	}

	acceptedEditors := hashset.New[string](
		"zed",
		"git",
		"vscode",
		"meld",
	)

	editor, err := utils.Prompt(reader, "Prefered diff viewer (zed, git, vscode, or meld): ")
	if err != nil {
		return err
	}

	if psqlPath != "" {
		cfg.Binaries.PSQL = psqlPath
	}
	if pgDumpPath != "" {
		cfg.Binaries.PGDump = pgDumpPath
	}
	if pgDumpAllPath != "" {
		cfg.Binaries.PGDumpAll = pgDumpAllPath
	}
	if editor != "" && acceptedEditors.Contains(editor) {
		cfg.Editor.Default = editor
	}
	if supabasePath != "" {
		cfg.Binaries.Supabase = supabasePath
	}

	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	utils.SuccessPrint("\nConfiguration saved successfully")
	return nil
}
