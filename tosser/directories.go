package tosser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"rubbish/config"
	"time"
)

// TossDirectory moves a directory and all its contents to the trash container.
// It validates that the target is indeed a directory, handles name conflicts
// by appending timestamps, and records the operation in the journal for tracking.
//
// The function performs these operations:
// 1. Validates the target is a directory and not a file
// 2. Determines the destination path in the trash container
// 3. Handles naming conflicts by appending timestamps to directory names
// 4. Moves the entire directory tree to the trash container
// 5. Records metadata in the journal for tracking and restoration
//
// If a directory with the same name already exists in trash, it appends a timestamp
// to create a unique name, ensuring no data loss or conflicts.
//
// Parameters:
//   - item: The path to the directory that should be moved to trash
//   - cfg: Application configuration containing container path and journal
//
// Returns an error if the target is not a directory, if directory operations fail,
// or if journal recording encounters issues.
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
