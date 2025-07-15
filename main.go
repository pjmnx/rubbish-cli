package main

import (
	"fmt"
	"os"
	"path/filepath"
	"rubbish/cleaner"
	"rubbish/config"
	"rubbish/restorer"
	"rubbish/status"
	"rubbish/tosser"
)

func loadConfig() (*config.Config, error) {
	// Define the paths for the configuration files
	defaultConfigPath := "/etc/rubbish/config.cfg"
	homedir, err := os.UserHomeDir()

	if err != nil {
		return nil, fmt.Errorf("error getting user home directory: %w", err)
	}
	userConfigPath := filepath.Join(homedir, ".config", "rubbish.cfg")

	// Load the configuration
	cfg, err := config.Load([]string{defaultConfigPath, userConfigPath})
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	return cfg, nil
}

type Command struct {
	Name        string
	Description string
	Run         func(args []string, cfg *config.Config) error
}

var commands = []Command{
	{
		Name:        "toss",
		Description: "Move files to the trash",
		Run:         tosser.Command, // Assuming tosser.Command is a function that handles the "toss" command
	},
	{
		Name:        "restore",
		Description: "Restore files from the trash",
		Run:         restorer.Command, // Assuming restorer.Command is a function that handles the "restore" command
	},
	{
		Name:        "cleanup",
		Description: "Clean up the trash",
		Run:         cleaner.Command,
	},
	{
		Name:        "status",
		Description: "Show the status of the trash",
		Run:         status.Command, // Assuming status.Command is a function that handles the "status" command
	},
	{
		Name:        "help",
		Description: "Show help information",
		Run:         showHelp, // Assuming helper.Command is a function that handles the "help" command
	},
}

func showHelp(args []string, cfg *config.Config) error {
	fmt.Println("Rubbish - A command-line trash management utility")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  rubbish <command> [args]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Printf("  %-10s %s\n", "toss", "Move files to the trash")
	fmt.Printf("  %-10s %s\n", "restore", "Restore files from the trash")
	fmt.Printf("  %-10s %s\n", "cleanup", "Clean up the trash")
	fmt.Printf("  %-10s %s\n", "status", "Show the status of the trash")
	fmt.Printf("  %-10s %s\n", "help", "Show help information")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  rubbish toss file.txt               # Move file.txt to trash")
	fmt.Println("  rubbish toss *.log                  # Move all .log files to trash")
	fmt.Println("  rubbish restore file.txt            # Restore file.txt from trash")
	fmt.Println("  rubbish status                      # Show trash status")
	fmt.Println("  rubbish cleanup                     # Clean up old files from trash")
	fmt.Println()
	fmt.Println("CONFIGURATION:")
	fmt.Printf("  Container Path:    %s\n", cfg.ContainerPath)
	fmt.Printf("  Swipe Time:        %d days\n", cfg.SwipeTime)
	fmt.Printf("  Max Retention:     %d days\n", cfg.MaxRetention)
	fmt.Printf("  Cleanup Interval:  %d days\n", cfg.CleanupInterval)
	fmt.Printf("  Log Retention:     %d days\n", cfg.LogRetention)
	fmt.Printf("  Notifications:     %t\n", cfg.Notification.Enabled)
	if cfg.Notification.Enabled {
		fmt.Printf("    Days in Advance: %d days\n", cfg.Notification.DaysInAdvance)
		fmt.Printf("    Timeout:         %d seconds\n", cfg.Notification.Timeout)
	}
	fmt.Println()
	fmt.Println("For more information, visit the project repository or documentation.")

	return nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
		os.Exit(1)
		return
	}

	// Validate if the container path exists
	if _, err := os.Stat(cfg.ContainerPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m Container path '%s' does not exist. Please check your configuration.\n", cfg.ContainerPath)
		if err := os.MkdirAll(cfg.ContainerPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m Failed to create container directory '%s': %v\n", cfg.ContainerPath, err)
			os.Exit(1)
			return
		}
		fmt.Printf("Created container directory: %s\n", cfg.ContainerPath)
	}

	//Validate command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: rubbish <command> [args]")
		fmt.Println("Use 'rubbish help' to see available commands.")
		os.Exit(1)
		return
	}

	cmdName := os.Args[1]
	for _, cmd := range commands {
		if cmd.Name == cmdName {
			err := cmd.Run(os.Args[2:], cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
				os.Exit(1)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m Unknown command '%s'. Use 'rubbish help' to see available commands.\n", cmdName)
	os.Exit(1)
}
