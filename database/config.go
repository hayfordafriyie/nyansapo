package database

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Provider string

const (
	ProviderMySQL      Provider = "mysql"
	ProviderPostgreSQL Provider = "postgres"
	ProviderMongoDB    Provider = "mongodb"
	ProviderCassandra  Provider = "cassandra"
)

type Config struct {
	Provider Provider
	URL      string
	Name     string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	TLS      bool
	ReadOnly bool
}

type SafeConfig struct {
	Provider    Provider
	Endpoint    string
	Database    string
	User        string
	ReadOnly    bool
	HasPassword bool
}

func LoadConfig() (Config, error) {
	_ = LoadDotEnv(".env")
	provider := Provider(strings.ToLower(strings.TrimSpace(os.Getenv("NYANSAPO_DB_PROVIDER"))))
	if provider == "" {
		return Config{}, errors.New("NYANSAPO_DB_PROVIDER is required")
	}
	switch provider {
	case ProviderMySQL, ProviderPostgreSQL, ProviderMongoDB, ProviderCassandra:
	default:
		return Config{}, fmt.Errorf("unsupported database provider %q", provider)
	}

	port := 0
	if value := strings.TrimSpace(os.Getenv("NYANSAPO_DB_PORT")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return Config{}, fmt.Errorf("NYANSAPO_DB_PORT must be a valid port")
		}
		port = parsed
	}
	readOnly := true
	if value := strings.TrimSpace(os.Getenv("NYANSAPO_DB_READ_ONLY")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("NYANSAPO_DB_READ_ONLY must be true or false")
		}
		readOnly = parsed
	}
	return Config{
		Provider: provider,
		URL:      strings.TrimSpace(os.Getenv("NYANSAPO_DB_URL")),
		Name:     strings.TrimSpace(os.Getenv("NYANSAPO_DB_NAME")),
		Host:     strings.TrimSpace(os.Getenv("NYANSAPO_DB_HOST")),
		Port:     port,
		User:     os.Getenv("NYANSAPO_DB_USER"),
		Password: os.Getenv("NYANSAPO_DB_PASSWORD"),
		Database: strings.TrimSpace(os.Getenv("NYANSAPO_DB_DATABASE")),
		TLS:      strings.EqualFold(strings.TrimSpace(os.Getenv("NYANSAPO_DB_TLS")), "true"),
		ReadOnly: readOnly,
	}, nil
}

func LoadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dotenv file: %w", err)
	}
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid dotenv entry on line %d", lineNumber+1)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if key == "" {
			return fmt.Errorf("empty dotenv key on line %d", lineNumber+1)
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set dotenv variable %s: %w", key, err)
			}
		}
	}
	return nil
}

func (c Config) Endpoint() string {
	if c.URL != "" {
		return c.URL
	}
	scheme := string(c.Provider)
	if c.Provider == ProviderPostgreSQL {
		scheme = "postgresql"
	}
	if c.Host == "" {
		return ""
	}
	host := c.Host
	if c.Port > 0 {
		host = fmt.Sprintf("%s:%d", host, c.Port)
	}
	return fmt.Sprintf("%s://%s@%s/%s", scheme, url.PathEscape(c.User), host, url.PathEscape(c.Database))
}

func (c Config) Safe() SafeConfig {
	return SafeConfig{
		Provider:    c.Provider,
		Endpoint:    c.Endpoint(),
		Database:    c.Database,
		User:        c.User,
		ReadOnly:    c.ReadOnly,
		HasPassword: c.Password != "",
	}
}

func (c Config) SQLDriverAndDSN() (string, string, error) {
	switch c.Provider {
	case ProviderMySQL:
		if c.URL != "" {
			return "mysql", c.URL, nil
		}
		if c.Host == "" || c.User == "" || c.Database == "" {
			return "", "", errors.New("mysql requires host, user, and database or NYANSAPO_DB_URL")
		}
		port := c.Port
		if port == 0 {
			port = 3306
		}
		return "mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", c.User, c.Password, c.Host, port, c.Database), nil
	case ProviderPostgreSQL:
		if c.URL != "" {
			return "pgx", c.URL, nil
		}
		if c.Host == "" || c.User == "" || c.Database == "" {
			return "", "", errors.New("postgres requires host, user, and database or NYANSAPO_DB_URL")
		}
		port := c.Port
		if port == 0 {
			port = 5432
		}
		sslMode := "disable"
		if c.TLS {
			sslMode = "require"
		}
		return "pgx", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", url.QueryEscape(c.User), url.QueryEscape(c.Password), c.Host, port, url.PathEscape(c.Database), sslMode), nil
	default:
		return "", "", fmt.Errorf("SQL connection is not supported for %s", c.Provider)
	}
}

func (c Config) DocumentURL() (string, error) {
	if c.URL != "" {
		return c.URL, nil
	}
	if c.Host == "" {
		return "", errors.New("document database requires host or NYANSAPO_DB_URL")
	}
	port := c.Port
	if port == 0 {
		if c.Provider == ProviderCassandra {
			port = 9042
		} else {
			port = 27017
		}
	}
	scheme := "mongodb"
	if c.Provider == ProviderCassandra {
		scheme = "cassandra"
	}
	credentials := ""
	if c.User != "" {
		credentials = url.QueryEscape(c.User)
		if c.Password != "" {
			credentials += ":" + url.QueryEscape(c.Password)
		}
		credentials += "@"
	}
	return fmt.Sprintf("%s://%s%s:%d/%s", scheme, credentials, c.Host, port, url.PathEscape(c.Database)), nil
}
