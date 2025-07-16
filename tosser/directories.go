package tosser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"rubbish/config"
	"time"
)

func TossDirectory(item string, cfg *config.Config) error {
	info, err := os.Stat(item)
	if err != nil {
		return fmt.Errorf("error getting file info for %s: %w", item, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot toss a file using TossDirectory: %s", item)
	}

	basename := filepath.Base(item)
	destination := path.Join(cfg.ContainerPath, basename)

	if _, err := os.Stat(destination); err == nil {
		destination = path.Join(cfg.ContainerPath, basename+"_"+time.Now().Format("20060102150405"))
		fmt.Printf("Directory already exists in the trash, renaming to: %s\n", destination)
	}

	go func() {
		if err := cfg.Journal.Add(basename, item, cfg.SwipeTime); err != nil {
			fmt.Fprintf(os.Stderr, "error adding directory to journal: %v\n", err)
		}
	}()

	if err := os.Rename(item, destination); err != nil {
		if errj := cfg.Journal.Delete(basename); errj != nil {
			return fmt.Errorf("error deleting journal entry for %s: %w", item, errj)
		}
		return fmt.Errorf("error moving directory %s to trash: %w", item, err)
	}
	fmt.Printf("Directory %s moved to trash at %s\n", item, destination)
	return nil
}
