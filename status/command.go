package status

import (
	"flag"
	"fmt"
	"path"
	"rubbish/config"
	"rubbish/journal"
	"strings"
)

var (
	Flags             = flag.NewFlagSet("status", flag.ExitOnError)
	globalLookup bool = false
	sizeOnly     bool = false
)

func init() {
	Flags.BoolVar(&globalLookup, "g", false, "Display rubbish status globally")
	Flags.BoolVar(&sizeOnly, "s", false, "Display the rubbish bin size only.")

	// configure the command options and flags
	Flags.Usage = func() {
		fmt.Println("Rubbish Status shows the current state of rubbish container.\n",
			"Usage:\n\n",
			"\trubbish status [options]\n\n",
			"Options:")
		Flags.PrintDefaults()
	}
}

// The status command is intended to show the current state of the rubbish,
// including the number of items, their retention times, and any other relevant
// metadata that can help users understand what is currently in the rubbish.
func Command(args []string, cfg *config.Config) error {

	var records []*journal.MetaData
	var err error

	totalSize, err := config.BinSize(cfg)

	if err != nil {
		return fmt.Errorf("error retrieving rubbish bin size: %w", err)
	}

	if sizeOnly {
		fmt.Printf("Rubbish bin size: %s\n", config.ReadableSize(uint64(totalSize)))
		return nil
	}

	records, err = retrieveJournalRecords(cfg)

	if err != nil {
		return fmt.Errorf("error retrieving rubbish items: %w", err)
	}

	if globalLookup {
		fmt.Println("Showing global rubbish status")
	}

	count := len(records)
	wipeables := 0

	if count == 0 {
		fmt.Println("No rubbish found.")
		return nil
	}

	println("Rubbish:")

	for _, record := range records {

		if !globalLookup {
			// Update the item name to reflect that is relative to the working directory
			record.Item = path.Join(strings.Replace(path.Dir(record.Origin), cfg.WorkingDir+"/", "", 1), record.Item)
		}

		if record.IsWipeable() {
			wipeables++
		}

		fmt.Println(" > " + String(record))
	}

	fmt.Printf("Total: %d | Wipable: %d | Bin Size: %s\n", count, wipeables, config.ReadableSize(uint64(totalSize)))

	return nil
}

func retrieveJournalRecords(cfg *config.Config) ([]*journal.MetaData, error) {
	var (
		records []*journal.MetaData
		err     error
	)

	switch {
	case globalLookup:
		records, err = cfg.Journal.List()
	case wipeableOnly:
		records, err = cfg.Journal.FilterWipeable()
	default:
		records, err = cfg.Journal.FilterPath(cfg.WorkingDir)
	}

	return records, err
}


func String(record *journal.MetaData) string {
	const msg = "%s | Tossed:%v | %s"

	remaining := record.RemainingTime()
	var remain_msg string

	switch {
	case remaining.Hours() > 24.0:
		remain_msg = fmt.Sprintf("WipeIn:%.01fd", remaining.Hours()/24.0)
	case remaining.Hours() > 0:
		remain_msg = fmt.Sprintf("WipeIn:%v", remaining)
	default:
		remain_msg = "Wipeable"
	}

	return fmt.Sprintf(msg, record.Item, record.TossElapsed(), remain_msg)
}
