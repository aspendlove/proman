package config

import (
	"encoding/json"
	"os"
)

// ConnectionParams holds the database connection details for a specific Supabase project.
type ConnectionParams struct {
	Password          string `json:"password"`
	Host              string `json:"host"`
	Port              string `json:"port"`
	User              string `json:"user"`
	DBName            string `json:"db_name"`
	SupabaseProjectID string `json:"supabase_project_id"`
}

// BinaryPaths holds the paths to various CLI tools.
type BinaryPaths struct {
	PSQL      string `json:"psql"`
	PGDumpAll string `json:"pg_dumpall"`
	PGDump    string `json:"pg_dump"`
	Supabase  string `json:"supabase"`
	Results   string `json:"results"`
}

// Config is the top-level configuration structure.
type Config struct {
	Connections map[string]ConnectionParams `json:"connections"`
	Binaries    BinaryPaths                 `json:"binaries"`
}

// Load reads the configuration from a JSON file.
func Load(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return a default config
			return &Config{
				Connections: make(map[string]ConnectionParams),
				Binaries:    BinaryPaths{},
			}, nil
		}
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to a JSON file.
func (c *Config) Save(filePath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// AddConnection adds or updates a connection profile.
func (c *Config) AddConnection(id string, params ConnectionParams) {
	c.Connections[id] = params
}

// GetConnection retrieves a connection profile by its ID.
// It returns the parameters and a boolean indicating if the key was found.
func (c *Config) GetConnection(id string) (ConnectionParams, bool) {
	params, found := c.Connections[id]
	return params, found
}

// RemoveConnection deletes a connection profile by its ID.
func (c *Config) RemoveConnection(id string) {
	delete(c.Connections, id)
}

// ListConnections returns a list of all connection IDs.
func (c *Config) ListConnections() []string {
	ids := make([]string, 0, len(c.Connections))
	for id := range c.Connections {
		ids = append(ids, id)
	}
	return ids
}

// SetBinaryPaths sets the entire binary paths configuration.
func (c *Config) SetBinaryPaths(paths BinaryPaths) {
	c.Binaries = paths
}

// GetBinaryPaths retrieves the binary paths configuration.
func (c *Config) GetBinaryPaths() BinaryPaths {
	return c.Binaries
}
