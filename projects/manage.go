package projects

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"proman/config"
	"strings"
	"text/tabwriter"
)

// prompt is a helper function to get user input from the console.
func prompt(reader *bufio.Reader, text string) (string, error) {
	fmt.Print(text)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// Register prompts the user for connection details and adds the new project to the config.
func Register(cfg *config.Config, configFile string, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("register command takes no arguments")
	}

	// --- Gather Information ---
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

	// NOTE: For a production-ready tool, you might want to use a library
	// to read the password without echoing it to the terminal.
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

	// --- Add and Save ---
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

// List prints a formatted table of all registered projects.
func List(cfg *config.Config, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("list command takes no arguments")
	}

	connectionIDs := cfg.ListConnections()
	if len(connectionIDs) == 0 {
		fmt.Println("No projects are registered yet. Use 'proman register' to add one.")
		return nil
	}

	// Initialize tabwriter for formatted output
	w := new(tabwriter.Writer)
	// The settings below are for nice padding and alignment.
	w.Init(os.Stdout, 0, 8, 2, ' ', 0)

	// Print table header
	fmt.Fprintln(w, "ID\tHOST\tUSER\tDATABASE")
	fmt.Fprintln(w, "--\t----\t----\t--------")

	// Print each connection's details
	for _, id := range connectionIDs {
		params, _ := cfg.GetConnection(id)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, params.Host, params.User, params.DBName)
	}

	// Flush the writer to print the table
	return w.Flush()
}

// Remove deletes a project from the configuration by its ID.
func Remove(cfg *config.Config, configFile string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("remove command expects exactly one argument: the project ID")
	}
	projectID := args[0]

	// Check if the project exists before trying to remove it.
	if _, found := cfg.GetConnection(projectID); !found {
		return fmt.Errorf("project with ID '%s' not found", projectID)
	}

	// Remove the connection from the config.
	cfg.RemoveConnection(projectID)

	// Save the updated configuration.
	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Successfully removed project '%s'.\n", projectID)
	return nil
}

// Login executes the 'supabase login' command using the configured binary path.
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

	// Connect the command's streams to the parent process's streams
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run '%s login': %w", supabasePath, err)
	}

	fmt.Println("\nLogin command finished.")
	return nil
}

// Init guides the user through setting up the required binary paths.
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

	resultsPath, err := prompt(reader, fmt.Sprintf("Path to results (current: %s): ", cfg.Binaries.Results))
	if err != nil {
		return err
	}

	supabasePath, err := prompt(reader, fmt.Sprintf("Path to supabase (current: %s): ", cfg.Binaries.Supabase))
	if err != nil {
		return err
	}

	// Update the config object only if the user provided new input.
	if psqlPath != "" {
		cfg.Binaries.PSQL = psqlPath
	}
	if pgDumpPath != "" {
		cfg.Binaries.PGDump = pgDumpPath
	}
	if pgDumpAllPath != "" {
		cfg.Binaries.PGDumpAll = pgDumpAllPath
	}
	if resultsPath != "" {
		cfg.Binaries.Results = resultsPath
	}
	if supabasePath != "" {
		cfg.Binaries.Supabase = supabasePath
	}

	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✅ Configuration saved successfully.")
	return nil
}
