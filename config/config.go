package config

import (
	"fmt"
	"path"
	"rubbish/journal"

	"github.com/go-ini/ini"
)

type Config struct {
	// Define your configuration fields here
	SwipeTime       int    `ini:"swipe_time"`
	ContainerPath   string `ini:"container_path"`
	MaxRetention    int    `ini:"max_retention"`
	CleanupInterval int    `ini:"cleanup_interval"`
	LogRetention    int    `ini:"log_retention"`
	Notification    struct {
		Enabled       bool `ini:"enabled"`
		DaysInAdvance int  `ini:"days_in_advance"`
		Timeout       int  `ini:"timeout"`
	} `ini:"notifications"`
	Journal *journal.Journal
}

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
