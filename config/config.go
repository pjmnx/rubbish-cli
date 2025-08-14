package config

import (
	"fmt"
	"math/bits"
	"os"
	"path"
	"path/filepath"
	"rubbish/journal"
	"strings"

	"github.com/go-ini/ini"
)

// Config represents the application configuration loaded from INI files.
// It contains all settings for the rubbish trash management utility,
// including file retention policies, notification settings, and paths.
// The configuration supports loading from multiple files with user overrides.
type Config struct {
	// WipeoutTime is the default number of days files remain in trash before
	// they become eligible for permanent deletion
	WipeoutTime int `ini:"wipeout_time"`

	// ContainerPath is the absolute path where trashed files are stored
	ContainerPath string `ini:"container_path"`

	// MaxRetention is the maximum number of days any file can remain in trash
	// regardless of individual wipeout time settings
	MaxRetention int `ini:"max_retention"`

	// CleanupInterval is how often (in days) the cleanup process should run
	// to remove expired files from trash
	CleanupInterval int `ini:"cleanup_interval"`

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

	WorkingDir string // workingDir is the current working directory of the application
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
		WipeoutTime:     30,
		ContainerPath:   ".local/share/rubbish",
		MaxRetention:    365,
		CleanupInterval: 3,
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

	config.ContainerPath = NormalizePath(config.ContainerPath)

	config.Journal = &journal.Journal{
		Path: path.Join(config.ContainerPath, ".journal"),
	}

	if err := config.Journal.Load(); err != nil {
		return nil, fmt.Errorf("failed to load journal: %w", err)
	}

	config.WorkingDir, err = os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current working directory: %w", err)
	}

	return config, nil
}

// Expands the user's home directory if it's a relative path and returns the absolute path to the container directory.
func NormalizePath(container_path string) string {
	if path.IsAbs(container_path) {
		return container_path
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return container_path
	}

	if container_path[0] == '~' {
		container_path = container_path[1:]
	}

	return path.Join(userHomeDir, container_path)
}

func BinSize(cfg *Config) (int64, error) {
	var size int64
	err := filepath.Walk(cfg.ContainerPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, ".journal") {
			return nil
		}

		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error calculating rubbish size: %v\n", err)
		return 0, err
	}
	return size, nil
}

func ReadableSize(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%d bytes", size)
	}

	base := uint(bits.Len64(size) / 10)
	val := float64(size) / float64(uint64(1<<(base*10)))

	return fmt.Sprintf("%.1f %cB", val, " KMGTPE"[base])
}
