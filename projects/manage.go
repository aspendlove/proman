package projects

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"strings"
	"text/tabwriter"

	"github.com/ugurcsen/gods-generic/sets/hashset"
)

func prompt(reader *bufio.Reader, text string) (string, error) {
	fmt.Print(text)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func Register(cfg *config.Config, configFile string, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("register command takes no arguments")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Registering a new Supabase project connection.")

	projectID, err := prompt(reader, "Enter a unique ID for this project: ")
	if err != nil {
		return err
	}
	if _, found := cfg.GetConnection(projectID); found {
		return fmt.Errorf("project with ID '%s' already exists", projectID)
	}

	host, err := prompt(reader, "Enter Host: ")
	if err != nil {
		return err
	}

	port, err := prompt(reader, "Enter Port (default: 5432): ")
	if err != nil {
		return err
	}
	if port == "" {
		port = "5432"
	}

	user, err := prompt(reader, "Enter User: ")
	if err != nil {
		return err
	}

	password, err := prompt(reader, "Enter Password: ")
	if err != nil {
		return err
	}

	dbName, err := prompt(reader, "Enter Database Name (default: postgres): ")
	if err != nil {
		return err
	}
	if dbName == "" {
		dbName = "postgres"
	}

	supabaseProjectID, err := prompt(reader, "Enter Supabase Project ID (e.g., ref_...): ")
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

	fmt.Printf("\nSuccessfully registered project '%s'.\n", projectID)
	return nil
}

func List(cfg *config.Config, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("list command takes no arguments")
	}

	connectionIDs := cfg.ListConnections()
	if len(connectionIDs) == 0 {
		fmt.Println("No projects are registered yet. Use 'proman register' to add one.")
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

	fmt.Printf("Successfully removed project '%s'.\n", projectID)
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

	fmt.Printf("Running `%s login`...\n", supabasePath)
	fmt.Println("Please follow the prompts from the Supabase CLI.")

	cmd := exec.Command(supabasePath, "login")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run '%s login': %w", supabasePath, err)
	}

	fmt.Println("\nLogin command finished.")
	return nil
}

func Init(cfg *config.Config, configFile string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Configure Binary Paths ---")
	fmt.Println("Please provide the absolute paths for the following tools.")
	fmt.Println("If a tool is already in your system's PATH, you can just enter its name (e.g., 'psql').")

	psqlPath, err := prompt(reader, fmt.Sprintf("Path to psql (current: %s): ", cfg.Binaries.PSQL))
	if err != nil {
		return err
	}

	pgDumpPath, err := prompt(reader, fmt.Sprintf("Path to pg_dump (current: %s): ", cfg.Binaries.PGDump))
	if err != nil {
		return err
	}

	pgDumpAllPath, err := prompt(reader, fmt.Sprintf("Path to pg_dumpall (current: %s): ", cfg.Binaries.PGDumpAll))
	if err != nil {
		return err
	}

	supabasePath, err := prompt(reader, fmt.Sprintf("Path to supabase (current: %s): ", cfg.Binaries.Supabase))
	if err != nil {
		return err
	}

	acceptedEditors := hashset.New[string](
		"zed",
		"git",
		"vscode",
		"meld",
	)

	editor, err := prompt(reader, "Prefered Diff Viewer (zed, git, vscode, or meld): ")
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

	fmt.Println("\nConfiguration saved successfully.")
	return nil
}
