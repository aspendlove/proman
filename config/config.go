package config

import (
	"encoding/json"
	"os"
)

type ConnectionParams struct {
	Password          string `json:"password"`
	Host              string `json:"host"`
	Port              string `json:"port"`
	User              string `json:"user"`
	DBName            string `json:"db_name"`
	SupabaseProjectID string `json:"supabase_project_id"`
}

type BinaryPaths struct {
	PSQL      string `json:"psql"`
	PGDumpAll string `json:"pg_dumpall"`
	PGDump    string `json:"pg_dump"`
	Supabase  string `json:"supabase"`
	Results   string `json:"results"`
}

type Config struct {
	Connections map[string]ConnectionParams `json:"connections"`
	Binaries    BinaryPaths                 `json:"binaries"`
}

func Load(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
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

func (c *Config) Save(filePath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (c *Config) AddConnection(id string, params ConnectionParams) {
	c.Connections[id] = params
}

func (c *Config) GetConnection(id string) (ConnectionParams, bool) {
	params, found := c.Connections[id]
	return params, found
}

func (c *Config) RemoveConnection(id string) {
	delete(c.Connections, id)
}

func (c *Config) ListConnections() []string {
	ids := make([]string, 0, len(c.Connections))
	for id := range c.Connections {
		ids = append(ids, id)
	}
	return ids
}

func (c *Config) SetBinaryPaths(paths BinaryPaths) {
	c.Binaries = paths
}

func (c *Config) GetBinaryPaths() BinaryPaths {
	return c.Binaries
}
