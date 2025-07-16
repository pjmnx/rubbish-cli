package tosser

import (
	"flag"
	"fmt"
	"os"
	"rubbish/config"
)

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
