package status

import (
	"flag"
	"fmt"
	"path"
	"rubbish/config"
	"rubbish/journal"
	"strings"
	"time"
)

var Flags = flag.NewFlagSet("status", flag.ExitOnError)

func init() {
	Flags.Bool("global", false, "Display global status of the trash")
	Flags.Bool("g", false, "Display global status of the trash (alias for --global)")

	// configure the command options and flags
	Flags.Usage = func() {
		fmt.Println("Usage: rubbish status [options]")
		fmt.Println("Options:")
		Flags.PrintDefaults()
	}
}

func isGlobalStatus() bool {
	// Check if the global flag is set
	return (Flags.Lookup("global") != nil && Flags.Lookup("global").Value.String() == "true") ||
		(Flags.Lookup("g") != nil && Flags.Lookup("g").Value.String() == "true")
}

func Command(args []string, cfg *config.Config) error {
	// The status command is intended to show the current state of the trash,
	// including the number of items, their retention times, and any other relevant
	// metadata that can help users understand what is currently in the trash.

	if cfg.Journal == nil {
		return fmt.Errorf("journal is not initialized, cannot show status")
	}

	if !Flags.Parsed() {
		err := Flags.Parse(args)
		if err != nil {
			return fmt.Errorf("error parsing flags: %w", err)
		}
	}

	// If the global flag is set, show the global status of the trash
	// Otherwise, show the local status
	globalFlag := isGlobalStatus()

	var records []*journal.MetaData
	var err error

	if globalFlag {
		fmt.Println("Showing global status of the trash...")
		records, err = cfg.Journal.GetAllItems()
		if err != nil {
			return fmt.Errorf("error retrieving global items from journal: %w", err)
		}
	} else {
		records, err = cfg.Journal.GetContainerItems(cfg.WorkingDir)
		if err != nil {
			return fmt.Errorf("error retrieving local items from journal: %w", err)
		}
	}

	fmt.Printf("Number of items: %d\n", len(records))
	if len(records) == 0 {
		fmt.Println("No items in the trash.")
		return nil
	}

	toWipeCount := 0
	// Count the number of items that are ready to be wiped out
	go func(records []*journal.MetaData, count *int) {
		for _, record := range records {
			daysSinceTossed := int(time.Since(time.Unix(record.TossedTime, 0)).Hours() / 24)
			if record.WipeoutTime-daysSinceTossed <= 0 {
				*count++
			}
		}
	}(records, &toWipeCount)

	var message string

	for _, record := range records {
		if !globalFlag {
			message = fmt.Sprintf("Rubbish: %s, Tossed Time: %s, Days to Wipe: %d\n",
				strings.Replace(strings.Replace(record.Origin, path.Base(record.Origin), record.Item, 1), cfg.WorkingDir+"/", "", 1),
				time.Since(time.Unix(record.TossedTime, 1)).String(),
				record.WipeoutTime-int(time.Since(time.Unix(record.TossedTime, 0)).Hours()/24))
		} else {
			message = fmt.Sprintf("Rubbish: %s, Tossed Time: %s, Days to Wipe: %d\n",
				record.Item,
				time.Since(time.Unix(record.TossedTime, 0)).String(),
				record.WipeoutTime-int(time.Since(time.Unix(record.TossedTime, 0)).Hours()/24))
		}
		fmt.Print(message)
	}

	fmt.Printf("Total items ready to be wiped out: %d\n", toWipeCount)

	return nil
}
