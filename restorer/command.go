package restorer

import (
	"flag"
	"fmt"
	"os"
	"path"
	"rubbish/config"
	"rubbish/journal"
	"slices"
)

var (
	Flags         = flag.NewFlagSet("restore", flag.ExitOnError)
	override bool = false
	silent   bool = false
)

func init() {
	Flags.BoolVar(&override, "override", false, "Override existing files during restoration")
	// Flags.BoolVar(&override, "o", false, "Override existing files during restoration (alias for --override)")
	Flags.BoolVar(&silent, "silent", false, "Suppress output messages")
	// Flags.BoolVar(&silent, "s", false, "Suppress output messages (alias for --silent)")

	Flags.Usage = func() {
		fmt.Println("Usage: rubbish restore [options] <file1> <file2> ...")
		fmt.Println("Options:")
		Flags.PrintDefaults()
	}
}

func Command(args []string, cfg *config.Config) error {
	// Implement the logic to restore files from the trash
	if !Flags.Parsed() {
		return fmt.Errorf("error parsing flags")
	}

	if len(Flags.Args()) == 0 {
		return fmt.Errorf("no files specified to restore")
	}

	if override {
		fmt.Println("Override mode enabled. Existing files will be replaced.")
	}

	if silent {
		fmt.Println("Silent mode enabled. No output will be displayed.")
	}

	local_rubbish, err := cfg.Journal.FilterPath(cfg.WorkingDir)

	if err != nil {
		return fmt.Errorf("error retrieving local rubbish: %v", err)
	}

	for _, file := range Flags.Args() {
		if file == "" {
			return fmt.Errorf("no files specified to restore")
		}

		//Validate if the file exists in the local rubbish
		if !slices.ContainsFunc(local_rubbish, func(record *journal.MetaData) bool {
			return record.Item == file
		}) {
			fmt.Printf("File %s doesn't belong to this directory rubbish.\n", file)
			continue
		}

		// Find the record in the local rubbish
		record_index := slices.IndexFunc(local_rubbish, func(item *journal.MetaData) bool {
			return item.Item == file
		})

		if record_index < 0 {
			fmt.Printf("File %s not found in the trash.\n", file)
			continue
		}

		record := local_rubbish[record_index]
		original_file := path.Base(record.Origin)

		// Check if a file with the same name exists in the current directory
		if _, err := os.Stat(original_file); err == nil && !override {
			if !silent {
				fmt.Printf("File %s restoring to %s and already exists in the current directory. Use --override to replace it.\n", file, original_file)
			}
			continue
		}

		// Restore the file
		if err := os.Rename(path.Join(cfg.ContainerPath, record.Item), original_file); err != nil {
			return fmt.Errorf("error restoring file %s: %v", file, err)
		}

		if err := cfg.Journal.Delete(record.Item); err != nil {
			return fmt.Errorf("error deleting journal record for file %s: %v", file, err)
		}

		fmt.Println("Restoring file:", file)
	}

	return nil
}
