package tosser

import (
	"flag"
	"fmt"
	"os"
	"rubbish/config"
)

// Command handles the "toss" command which moves files and directories to trash.
// It processes command-line arguments to determine retention time and validates
// each file before moving it to the trash container. The function supports
// custom retention periods via the --retention flag.
//
// The command performs the following operations:
// 1. Parses command-line flags for retention time override
// 2. Validates that files/directories exist and are accessible
// 3. Determines whether each item is a file or directory
// 4. Delegates to appropriate tossing functions
// 5. Updates the journal with metadata about tossed items
// 6. Provides feedback about the operation results
//
// Parameters:
//   - args: Command-line arguments passed to the toss command
//   - cfg: Application configuration containing default settings and journal
//
// Returns an error if no files are specified, if any file cannot be accessed,
// or if the tossing operation fails for any item.
func Command(args []string, cfg *config.Config) error {
	fs := flag.NewFlagSet("toss", flag.ExitOnError)

	fs.IntVar(&cfg.SwipeTime, "retention", cfg.SwipeTime, "Time to wait before swiping the file(s) from the trash (in days)")

	fs.Parse(args)

	if len(fs.Args()) == 0 {
		return fmt.Errorf("no files or directory specified to toss")
	}

	for _, file := range fs.Args() {
		if file == "" {
			return fmt.Errorf("no files specified to toss")
		}

		fileinfo, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("error getting file info for %s: %w", file, err)
		}

		var tosser func(string, *config.Config) error

		if fileinfo.IsDir() {
			tosser = TossDirectory
		} else {
			tosser = TossFile
		}
		err = tosser(file, cfg)
		if err != nil {
			return fmt.Errorf("error tossing file %s: %w", file, err)
		}
	}

	count, err := cfg.Journal.Count()
	if err != nil {
		fmt.Printf("error getting journal count: %v\n", err.Error())
	} else {
		fmt.Printf("Total tossed files (local): %d\n", count)
	}

	fmt.Fprintf(os.Stdout, "\033[32mTossing\033[0m files %s to trash with a wait time of %d days.\n", fs.Args(), cfg.SwipeTime)
	return nil
}
