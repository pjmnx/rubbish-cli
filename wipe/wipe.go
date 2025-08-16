package wipe

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"rubbish/config"
	"rubbish/journal"
	"slices"
)

var (
	Flags           *flag.FlagSet = flag.NewFlagSet("wipe", flag.ExitOnError)
	forceWipeout    bool          = false // completeWipeout indicates whether to perform a complete wipe of the rubbish container
	autoAcknowledge bool          = false // autoAcknowledge indicates whether to automatically acknowledge the wipe operation by the user
	globalWipeout   bool          = false // globalWipeout indicates whether to perform a global wipe of all items in the journal
)

func init() {
	Flags.Usage = func() {
		println("Rubbish Wipe allows the user to remove items from the rubbish container.\n")
		println("Usage:\n",
			"\trubbish wipe [options] [<file>]\n")

		println("Options:")
		Flags.PrintDefaults()
	}
	Flags.BoolVar(&forceWipeout, "f", false, "Force wipe of the rubbish regardless of their WipeoutTime (default: false).")
	Flags.BoolVar(&autoAcknowledge, "y", false, "Automatically acknowledge the wipe operation (default: false).")
	Flags.BoolVar(&globalWipeout, "g", false, "Perform a global wipe of all items in the journal (default: false).")
}

func Command(args []string, cfg *config.Config) error {
	records, err := getRecords(cfg, globalWipeout, forceWipeout)

	if err != nil {
		return fmt.Errorf("error retrieving items from journal: %v", err)
	}

	if len(records) == 0 {
		fmt.Println("\033[31mNo valid items found to wipe.\033[0m")
		return nil
	}

	if len(Flags.Args()) > 0 {
		if err := wipeSelectedFiles(records, Flags.Args(), cfg); err != nil {
			return fmt.Errorf("error wiping files %s: %v", Flags.Args(), err)
		}
		return nil
	}

	if err := wipeAllFiles(records, cfg); err != nil {
		return fmt.Errorf("error wiping all files: %v", err)
	}

	return nil
}

// confirmWipe prompts the user for confirmation before wiping an item, unless autoAcknowledge is true.
func confirmWipe(item string) (bool, error) {
	if autoAcknowledge {
		return true, nil
	}
	fmt.Printf("Are you sure you want to wipe '%s'? [y/N]: ", item)
	var response string
	_, err := fmt.Scanln(&response)

	if err != nil {
		return false, err
	}
	if response == "y" || response == "Y" {
		return true, nil
	}
	return false, nil
}

func getRecords(cfg *config.Config, global bool, ignoreWipeTime bool) ([]*journal.MetaData, error) {
	var (
		records []*journal.MetaData
		result  []*journal.MetaData
		err     error
	)

	if global {
		fmt.Println("Performing global wipeout of all items in the journal...")
		records, err = cfg.Journal.List()
		if err != nil {
			return nil, fmt.Errorf("error retrieving all items from journal: %v", err)
		}

	} else {
		fmt.Println("Performing local wipeout of items in the rubbish container...")
		records, err = cfg.Journal.FilterPath(cfg.WorkingDir)
		if err != nil {
			return nil, fmt.Errorf("error retrieving container items from journal: %v", err)
		}
	}

	if !ignoreWipeTime {
		for _, record := range records {
			if !record.IsWipeable() {
				continue
			}
			result = append(result, record)
		}
	} else {
		result = records
	}

	return result, nil
}

func wipeSelectedFiles(records []*journal.MetaData, files []string, cfg *config.Config) error {
	for _, file := range files {
		i := slices.IndexFunc(records, func(record *journal.MetaData) bool {
			return record.Item == file
		})

		if i == -1 {
			return fmt.Errorf("file %s not found in the rubbish", file)
		}

		wipeConfirmed, err := confirmWipe(records[i].Item)
		if err != nil {
			return fmt.Errorf("error confirming wipe for %s: %v", records[i].Item, err)
		}
		if !wipeConfirmed {
			fmt.Printf("Skipping %s as per user confirmation.\n", records[i].Item)
			continue
		}

		if err := executeWipe(records[i], cfg); err != nil {
			fmt.Printf("Error wiping %s: %v\n", records[i].Item, err)
		}
	}

	return nil
}

func executeWipe(record *journal.MetaData, cfg *config.Config) error {
	if record == nil {
		return fmt.Errorf("record is nil, cannot wipe")
	}
	if cfg == nil {
		return fmt.Errorf("config is nil, cannot wipe")
	}

	rubbishFile := filepath.Join(cfg.ContainerPath, record.Item)

	if err := cfg.Journal.Delete(record.Item); err != nil {
		return fmt.Errorf("error deleting record for %s: %v", record.Item, err)
	}

	if err := os.RemoveAll(rubbishFile); err != nil {
		cfg.Journal.AddRecord(record) // Re-add the record if removal fails
		return fmt.Errorf("error removing rubbish file %s: %v", rubbishFile, err)
	}

	fmt.Printf("Wiped %s successfully.\n", record.Item)
	return nil
}

func wipeAllFiles(records []*journal.MetaData, cfg *config.Config) error {

	for _, record := range records {

		wipeConfirmed, err := confirmWipe(record.Item)
		if err != nil {
			return fmt.Errorf("error confirming wipe for %s: %v", record.Item, err)
		}

		if !wipeConfirmed {
			fmt.Printf("Skipping %s as per user confirmation.\n", record.Item)
			continue
		}

		if err := executeWipe(record, cfg); err != nil {
			fmt.Printf("Error wiping %s: %v\n", record.Item, err)
		}
	}

	return nil
}
