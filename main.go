package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"rubbish/config"
	"rubbish/info"
	"rubbish/restorer"
	"rubbish/status"
	"rubbish/tosser"
	"rubbish/wipe"
	"runtime/debug"
	"slices"
	"strings"
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

	// Action is the function that implements the command's functionality.
	// It receives command arguments and configuration, returning an error
	// if the operation fails.
	Action func(args []string, cfg *config.Config) error

	Options *flag.FlagSet // Optional flags for the command
}

// commands defines all available commands in the rubbish utility.
// Each command is mapped to its corresponding implementation function
// from the appropriate module (tosser, restorer, cleaner, status).
var (
	cmdToss *Command = &Command{
		Name:        "toss",
		Description: "Move files to the trash",
		Action:      tosser.Command, // Assuming tosser.Command is a function that handles the "toss" command
		Options:     tosser.Flags,   // Optional
	}
	cmdRestore *Command = &Command{
		Name:        "restore",
		Description: "Restore files from the trash",
		Action:      restorer.Command, // Assuming restorer.Command is a function that handles the "restore" command
		Options:     restorer.Flags,
	}
	cmdWipe *Command = &Command{
		Name:        "wipe",
		Description: "Clean up the rubbish",
		Action:      wipe.Command, // Assuming cleaner.Command is a function that handles the "wipe" command
		Options:     wipe.Flags,   // Optional
	}
	// cmdStatus is the command for showing the status of the trash
	cmdStatus *Command = &Command{
		Name:        "status",
		Description: "Show the status of the trash",
		Action:      status.Command, // Assuming status.Command is a function that handles the "status" command
		Options:     status.Flags,
	}
	cmdInfo *Command = &Command{
		Name:        "info",
		Description: "Show information about a rubbish item",
		Action:      info.Command,
		Options:     info.Flags,
	}
	cmdHelp *Command = &Command{
		Name:        "help",
		Description: "Show help information",
		Action: func(args []string, cfg *config.Config) error {

			if len(args) == 0 {
				printGeneralHelp()
				return nil
			}

			index := slices.IndexFunc(commands, func(c *Command) bool {
				return c.Name == strings.ToLower(args[0])
			})

			if index == -1 {
				return fmt.Errorf("unknown command: %s", args[0])
			}

			// Display help information
			commands[index].Options.Usage()

			return nil
		}, // Assuming helper.Command is a function that handles the "help" command
		Options: flag.NewFlagSet("help", flag.ExitOnError), // No specific flags for help, but can be extended
	}

	commands    []*Command = []*Command{cmdToss, cmdRestore, cmdStatus, cmdInfo, cmdWipe}
	helpCommand *Command
)

func printGeneralHelp() {
	fmt.Print("Rubbish is a tool to manage your trash effectively.\n\n",
		"Usage:\n\n",
		"  rubbish <command> [options]\n\n",
		"Available commands:\n\n")

	for _, cmd := range commands {
		fmt.Printf("\t%s\t\t%s\n", cmd.Name, cmd.Description)
	}

	fmt.Println("\nUse \"rubbish help <command>\" for more information on a specific command.")
}

func displayVersion() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		fmt.Printf("Build Version: %s\n", info.Main.Version)
	} else {
		fmt.Printf("Build Version: %s\n", "unknown")
	}
}

func init() {
	flag.BoolFunc("version", "Show version information", func(s string) error {
		displayVersion()
		os.Exit(0)
		return nil
	})
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
	// if len(flag.Args()) < 1 {
	// 	printGeneralHelp()
	// 	os.Exit(0)
	// 	return
	// }

	if cmdHelp.Name == flag.Arg(0) {
		cmdHelp.Options.Parse(flag.Args()[1:])
		err := cmdHelp.Action(cmdHelp.Options.Args(), cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
			printGeneralHelp()
			os.Exit(2)
			return
		}
		return
	}

	if !slices.ContainsFunc(commands, func(c *Command) bool {
		return c.Name == flag.Arg(0)
	}) {
		if flag.Arg(0) == "" {
			fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m Unknown command\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m Unknown command '%s'\n\n", flag.Arg(0))
		}

		printGeneralHelp()
		os.Exit(1)
		return
	}

	notifyExistingWipeables(cfg) // Notify about wipeable items in the dumpster

	for _, cmd := range commands {
		if cmd.Name == flag.Arg(0) {
			cmd.Options.Parse(flag.Args()[1:])

			err := cmd.Action(cmd.Options.Args(), cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
				os.Exit(2)
				return
			}
		}
	}
}

func notifyExistingWipeables(cfg *config.Config) {
	if stats, err := cfg.Journal.FilterWipeable(); err == nil {
		fmt.Printf("\033[33;1mNotice:\033[0m\033[33m Wipeable items in dumpster: %d\033[0m\n", len(stats))
	}
}
