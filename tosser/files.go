package tosser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"rubbish/config"
	"time"
)

// TossFile moves a regular file to the trash container with metadata tracking.
// It validates that the target is indeed a file (not a directory), handles
// name conflicts by appending timestamps, and records the operation in the journal.
//
// The function performs these operations:
// 1. Validates the target is a file and not a directory
// 2. Determines the destination path in the trash container
// 3. Handles naming conflicts by appending timestamps
// 4. Moves the file to the trash container
// 5. Records metadata in the journal for tracking and restoration
//
// If a file with the same name already exists in trash, it appends a timestamp
// to create a unique name, ensuring no data loss.
//
// Parameters:
//   - file: The path to the file that should be moved to trash
//   - cfg: Application configuration containing container path and journal
//
// Returns an error if the target is not a file, if file operations fail,
// or if journal recording encounters issues.
func TossFile(item string, cfg *config.Config) error {
	info, err := os.Stat(item)
	if err != nil {
		return fmt.Errorf("error getting file info for %s: %w", item, err)
	}
	// Check if the item is a file
	if info.IsDir() {
		return fmt.Errorf("cannot toss a directory using TossFile: %s", item)
	}

	basename := filepath.Base(item)
	destination := path.Join(cfg.ContainerPath, basename)

	if _, err := os.Stat(destination); err == nil {
		destination = path.Join(cfg.ContainerPath, basename+"_"+time.Now().Format("20060102150405"))
		fmt.Printf("File already exists in the trash, renaming to: %s\n", destination)
	}

	// Record the file in the journal for tracking
	go func() {
		if err := cfg.Journal.AddFileByName(filepath.Base(destination), item, cfg.WipeoutTime); err != nil {
			fmt.Printf("error adding file to journal: %v\n", err)
		}
	}()

	if err := os.Rename(item, destination); err != nil {
		if errj := cfg.Journal.Delete(filepath.Base(destination)); errj != nil {
			return fmt.Errorf("error deleting journal entry for %s: %w", item, errj)
		}
		return fmt.Errorf("error moving file %s to trash: %w", item, err)
	}
	fmt.Printf("File %s moved to trash at %s\n", item, destination)
	return nil
}
