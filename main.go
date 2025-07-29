package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"rubbish/cleaner"
	"rubbish/config"
	"rubbish/restorer"
	"rubbish/status"
	"rubbish/tosser"
)

// loadConfig loads the application configuration from system and user configuration files.
// It attempts to load a system-wide configuration first, then appends user-specific
// configuration from the user's home directory, allowing for personalized overrides.
//
// The configuration loading follows this hierarchy:
// 1. System default: /etc/rubbish/config.cfg
// 2. User override: ~/.config/rubbish.cfg
//
// Returns a fully initialized Config struct with default values and user overrides
// applied, or an error if the user home directory cannot be determined or if
// configuration loading fails.
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

// Command represents a command that can be executed by the rubbish utility.
// It encapsulates the command name, description, and the function that
// implements the command's functionality. This structure enables a clean
// command dispatch system with consistent error handling.
type Command struct {
	// Name is the command identifier used on the command line
	Name string

	// Description is a brief explanation of what the command does,
	// displayed in help output
	Description string

	// Run is the function that implements the command's functionality.
	// It receives command arguments and configuration, returning an error
	// if the operation fails.
	Run func(args []string, cfg *config.Config) error

	Flags *flag.FlagSet // Optional flags for the command
}

// commands defines all available commands in the rubbish utility.
// Each command is mapped to its corresponding implementation function
// from the appropriate module (tosser, restorer, cleaner, status).
var commands []Command
var helpCommand *Command

func displayVersion() error {
	fmt.Println("Rubbish Trash Management Utility")
	fmt.Println("Version: 0.0.1")
	fmt.Println("Author: pjmnx (Paul J.)")
	fmt.Println("License: MIT")
	return nil
}

func init() {
	flag.Bool("version", false, "Show version information")
	flag.Bool("V", false, "Show version information (alias for --version)")

	commands = []Command{
		{
			Name:        "toss",
			Description: "Move files to the trash",
			Run:         tosser.Command, // Assuming tosser.Command is a function that handles the "toss" command
			Flags:       tosser.Flags,   // Optional
		},
		{
			Name:        "restore",
			Description: "Restore files from the trash",
			Run:         restorer.Command, // Assuming restorer.Command is a function that handles the "restore" command
			Flags:       restorer.Flags,
		},
		{
			Name:        "cleanup",
			Description: "Clean up the trash",
			Run:         cleaner.Command,                              // Assuming cleaner.Command is a function that handles the "cleanup" command
			Flags:       flag.NewFlagSet("cleanup", flag.ExitOnError), // Optional
		},
		{
			Name:        "status",
			Description: "Show the status of the trash",
			Run:         status.Command, // Assuming status.Command is a function that handles the "status" command
			Flags:       status.Flags,
		},
		{
			Name:        "help",
			Description: "Show help information",
			Run:         showHelp,                                  // Assuming helper.Command is a function that handles the "help" command
			Flags:       flag.NewFlagSet("help", flag.ExitOnError), // No specific flags for help, but can be extended
		},
	}
	helpCommand = &commands[4]

}

// showHelp displays comprehensive help information for the rubbish utility.
// It provides usage instructions, available commands, practical examples,
// and current configuration settings to help users understand and effectively
// use the trash management system.
//
// The help output includes:
// - Basic usage syntax
// - List of all available commands with descriptions
// - Practical examples for common operations
// - Current configuration values for reference
// - Additional resources for more information
//
// Parameters:
//   - args: Command-line arguments (unused in help display)
//   - cfg: Application configuration used to display current settings
//
// Returns nil as help display operations do not fail.
func showHelp(args []string, cfg *config.Config) error {
	fs := flag.NewFlagSet("help", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Println("Usage: rubbish <command> [options]")
		fmt.Println("Available commands:")
		for _, cmd := range commands {
			fmt.Printf("  %s: %s\n", cmd.Name, cmd.Description)
		}
		fmt.Println("\nUse 'rubbish <command> --help' for more information on a specific command.")
	}
	fs.Parse(args)

	for _, cmd := range commands {
		if fs.Arg(0) != "" && cmd.Name != fs.Arg(0) {
			fmt.Printf("Command: %s\nDescription: %s\n", cmd.Name, cmd.Description)
			if cmd.Flags != nil {
				fmt.Println("Usage:")
				cmd.Flags.PrintDefaults()
			}
			fmt.Println()
		}
	}
	return nil
}

// main is the entry point for the rubbish trash management utility.
// It orchestrates the entire application flow including configuration loading,
// directory validation, command parsing, and execution.
//
// The main function performs these key operations:
// 1. Loads configuration from system and user files
// 2. Validates and creates the trash container directory if needed
// 3. Parses command-line arguments to determine the requested operation
// 4. Dispatches to the appropriate command handler
// 5. Handles errors with colored output and appropriate exit codes
//
// The function ensures proper error handling and user feedback throughout
// the application lifecycle, using ANSI color codes for enhanced readability
// of error messages.
//
// Exit codes:
//   - 0: Successful operation
//   - 1: Configuration error, directory creation failure, invalid command,
//     or command execution error
func main() {
	flag.Parse()

	if flag.Lookup("version").Value.String() == "true" || flag.Lookup("V").Value.String() == "true" {
		displayVersion()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
		os.Exit(1)
		return
	}

	defer cfg.Journal.Close()

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
	if len(flag.Args()) < 1 {
		fmt.Println("Usage: rubbish <command> [args]")
		fmt.Println("Use 'rubbish help' to see available commands.")
		showHelp(nil, cfg)
		os.Exit(1)
		return
	}

	cmdName := flag.Arg(0)
	for _, cmd := range commands {
		if cmd.Name == cmdName {
			cmd.Flags.Parse(flag.Args()[1:])

			err := cmd.Run(cmd.Flags.Args(), cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m Unknown command '%s'\n", cmdName)
	fmt.Println("Use 'rubbish help' to see available commands.")
	helpCommand.Flags.PrintDefaults()
	os.Exit(1)
}
