package caldav

import (
	"fmt"
	"os"
)

// Config holds CalDAV connection parameters read from environment variables.
type Config struct {
	URL      string
	Username string
	Password string
}

// ConfigFromEnv reads CALDAV_URL, CALDAV_USERNAME, and CALDAV_PASSWORD from the
// environment. Returns an error if any required variable is missing.
func ConfigFromEnv() (*Config, error) {
	url := os.Getenv("CALDAV_URL")
	if url == "" {
		return nil, fmt.Errorf("CALDAV_URL environment variable is required")
	}
	username := os.Getenv("CALDAV_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("CALDAV_USERNAME environment variable is required")
	}
	password := os.Getenv("CALDAV_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("CALDAV_PASSWORD environment variable is required")
	}
	return &Config{
		URL:      url,
		Username: username,
		Password: password,
	}, nil
}
