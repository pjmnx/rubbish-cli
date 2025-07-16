package config

import (
	"fmt"
	"path"
	"rubbish/journal"

	"github.com/go-ini/ini"
)

// Config represents the application configuration loaded from INI files.
// It contains all settings for the rubbish trash management utility,
// including file retention policies, notification settings, and paths.
// The configuration supports loading from multiple files with user overrides.
type Config struct {
	// SwipeTime is the default number of days files remain in trash before
	// they become eligible for permanent deletion
	SwipeTime int `ini:"swipe_time"`

	// ContainerPath is the absolute path where trashed files are stored
	ContainerPath string `ini:"container_path"`

	// MaxRetention is the maximum number of days any file can remain in trash
	// regardless of individual swipe time settings
	MaxRetention int `ini:"max_retention"`

	// CleanupInterval is how often (in days) the cleanup process should run
	// to remove expired files from trash
	CleanupInterval int `ini:"cleanup_interval"`

	// LogRetention is the number of days to keep log files before removing them
	LogRetention int `ini:"log_retention"`

	// Notification contains settings for system notifications about pending deletions
	Notification struct {
		// Enabled determines whether notifications should be sent
		Enabled bool `ini:"enabled"`

		// DaysInAdvance is how many days before deletion to send notifications
		DaysInAdvance int `ini:"days_in_advance"`

		// Timeout is how long (in seconds) notifications should be displayed
		Timeout int `ini:"timeout"`
	} `ini:"notifications"`

	// Journal is the database instance used to track metadata for trashed items
	Journal *journal.Journal
}

// Load reads configuration from the specified INI file paths and initializes
// the configuration struct with default values and file overrides.
// It loads a system-wide configuration first, then appends user-specific
// configuration, allowing users to override system defaults.
//
// The function also initializes the journal database for tracking trashed items
// and ensures it's ready for use.
//
// Parameters:
//   - paths: A slice containing paths to configuration files. The first path
//     should be the system default, the second should be user-specific.
//
// Returns a fully initialized Config struct with journal database ready,
// or an error if configuration loading, mapping, or journal initialization fails.
func Load(paths []string) (*Config, error) {
	cfg, err := ini.Load(paths[0])

	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	// Load complemetary file from user's home directory
	err = cfg.Append(paths[1])
	if err != nil {
		return nil, fmt.Errorf("failed to append user configuration: %w", err)
	}

	// Creating a default configuration if the file is empty
	config := &Config{
		SwipeTime:       30,
		ContainerPath:   "~/.local/share/rubbish",
		MaxRetention:    365,
		CleanupInterval: 7,
		LogRetention:    30,
		Notification: struct {
			Enabled       bool `ini:"enabled"`
			DaysInAdvance int  `ini:"days_in_advance"`
			Timeout       int  `ini:"timeout"`
		}{
			Enabled:       false,
			DaysInAdvance: 7,
			Timeout:       5,
		},
	}
	// Map the configuration file to the Config struct
	err = cfg.MapTo(config)
	if err != nil {
		return nil, fmt.Errorf("failed to map configuration: %w", err)
	}

	config.Journal = &journal.Journal{
		Path: path.Join(config.ContainerPath, ".journal"),
	}

	if err := config.Journal.Load(); err != nil {
		return nil, fmt.Errorf("failed to load journal: %w", err)
	}

	return config, nil
}
