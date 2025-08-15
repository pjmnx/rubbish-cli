package info

import (
	"flag"
	"fmt"
	"math"
	"path/filepath"
	"rubbish/config"
	"rubbish/journal"
	"slices"
	"time"
)

var (
	Flags      *flag.FlagSet = flag.NewFlagSet("info", flag.ExitOnError)
	byPosition int           = 0
)

func init() {
	Flags.IntVar(&byPosition, "p", 0, "The position of the item (1-based).")

	Flags.Usage = func() {
		fmt.Println("Rubbish info shows the rubbish item details.\n",
			"Usage:\n\n",
			"\trubbish info <item>\n",
			"\trubbish info -p=<position>\n\n",
			"Options:")
		Flags.PrintDefaults()
	}
}

func Command(args []string, cfg *config.Config) error {
	var (
		record *journal.MetaData
		err    error
	)

	if byPosition != 0 {
		record, err = retrieveByPosition(byPosition, cfg)
		if err != nil {
			return err
		}
	} else {
		if len(args) < 1 {
			return fmt.Errorf("item name is required")
		}
		record, err = retrieveByName(args[0], cfg)
		if err != nil {
			return err
		}
	}

	if record == nil {
		return fmt.Errorf("item not found: %s", args[0])
	}

	ttime := time.Unix(record.TossedTime, 0)
	wtime := ttime.Add(time.Duration(record.WipeoutTime*24) * time.Hour)
	rtime := record.RemainingTime()

	fmt.Printf("Item: %s\n", record.Item)
	fmt.Printf("Origin: %s\n", record.Origin)
	fmt.Printf("Tossed At: %v\n", ttime)
	fmt.Printf("Wipeable At: %s\n", wtime.Format(time.DateOnly)) //time.Date(wtime.Year(), wtime.Month(), wtime.Day(), 0, 0, 0, 0, wtime.Location()))

	if rtime >= 0 {
		fmt.Printf("Remaining: %s\n", rtime.Round(time.Second))
	} else {
		fmt.Printf("Remaining: %s (overdue)\n", (rtime * -1).Round(time.Second))
	}

	return nil
}

func retrieveByPosition(byPosition int, cfg *config.Config) (*journal.MetaData, error) {
	list, err := cfg.Journal.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	i := int(math.Abs(float64(byPosition)))

	if i > len(list) {
		return nil, fmt.Errorf("invalid item position: %d", byPosition)
	}

	i-- // Convert to zero-based index

	if byPosition < 0 {
		slices.Reverse(list)
		record := list[i]
		return record, nil
	}

	record := list[i]
	return record, nil

}

func retrieveByName(name string, cfg *config.Config) (*journal.MetaData, error) {
	itemName := filepath.Base(name)
	// Fetch and display item details using itemName

	record, err := cfg.Journal.Get(itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	return record, nil
}
