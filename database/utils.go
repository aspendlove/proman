package database

import (
	"fmt"
	"proman/config"
)

func FormatRemoteConnectionString(connection config.ConnectionParams) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		connection.User, connection.Password, connection.Host, connection.Port, connection.DBName,
	)
}
