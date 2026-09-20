package database

import (
	"fmt"
	"net/url"
)

func connectionDriverAndDSN(config Config) (string, string, error) {
	if config.Provider == ProviderPostgreSQL && config.URL == "" {
		if config.Host == "" || config.User == "" || config.Database == "" {
			return "", "", fmt.Errorf("postgres requires host, user, and database or NYANSAPO_DB_URL")
		}
		port := config.Port
		if port == 0 {
			port = 5432
		}
		sslMode := "disable"
		if config.TLS {
			sslMode = "require"
		}
		return "pgx", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			url.QueryEscape(config.User), url.QueryEscape(config.Password), config.Host,
			port, url.PathEscape(config.Database), sslMode), nil
	}
	return config.SQLDriverAndDSN()
}
