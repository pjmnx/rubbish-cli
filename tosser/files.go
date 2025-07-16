package tosser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"rubbish/config"
	"time"
)

func TossFile(file string, cfg *config.Config) error {
	info, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("error getting file info for %s: %w", file, err)
	}
	if info.IsDir() {
		return fmt.Errorf("cannot toss a directory using TossFile: %s", file)
	}

	basename := filepath.Base(file)
	destination := path.Join(cfg.ContainerPath, basename)

	if _, err := os.Stat(destination); err == nil {
		destination = path.Join(cfg.ContainerPath, basename+"_"+time.Now().Format("20060102150405"))
		fmt.Printf("File already exists in the trash, renaming to: %s\n", destination)
	}

	go func() {
		if err := cfg.Journal.Add(filepath.Base(destination), file, cfg.SwipeTime); err != nil {
			fmt.Fprintf(os.Stderr, "error adding file to journal: %v\n", err)
		}
	}()

	if err := os.Rename(file, destination); err != nil {
		if errj := cfg.Journal.Delete(filepath.Base(destination)); errj != nil {
			return fmt.Errorf("error deleting journal entry for %s: %w", file, errj)
		}
		return fmt.Errorf("error moving file %s to trash: %w", file, err)
	}
	fmt.Printf("File %s moved to trash at %s\n", file, destination)
	return nil
}

// func TossFileOld(file string, cfg *config.Config) error {
// 	// Implement the logic to move the file to the trash
// 	// This is a placeholder implementation

// 	dir, err := getDirectory(file)
// 	if err != nil {
// 		return fmt.Errorf("error getting directory for file %s: %w", file, err)
// 	}
// 	fmt.Printf("Tossing file: %s from directory: %s\n", file, dir)

// 	wd, err := os.Getwd()
// 	if err != nil {
// 		return fmt.Errorf("error getting current working directory: %w", err)
// 	}

// 	// file_data := config.GenerateMetadata(file, os.Getuid(), wd, cfg.ContainerPath, cfg.SwipeTime)
// 	fmt.Printf("Tossing file: %s with wait time: %d days\n", file, cfg.SwipeTime)

// 	dst_name := cfg.ContainerPath + "/" + file

// 	if _, err := os.Stat(dst_name); err == nil {
// 		// return fmt.Errorf("file %s already exists in the trash", dst_name)
// 		dst_name = path.Join(cfg.ContainerPath, file+"_"+time.Now().Format("20060102150405"))
// 		fmt.Printf("File already exists in the trash, renaming to: %s\n", dst_name)
// 	}

// 	// Generate metadata for the file being tossed
// 	toss_data := journal.GenerateMetadata(filepath.Base(dst_name), file, os.Getuid(), wd, cfg.ContainerPath, cfg.SwipeTime)

// 	if toss_data == nil {
// 		return fmt.Errorf("error generating metadata for file %s", file)
// 	}

// 	// go cfg.Journals.Add(toss_data) // Add metadata to the journal asynchronously
// 	go func() {
// 		if err := cfg.Journals.Add(toss_data); err != nil {
// 			fmt.Fprintf(os.Stderr, "Error adding metadata to journal: %v\n", err)
// 		}
// 	}()

// 	err = os.Rename(file, dst_name) // Move file to the trash directory
// 	if err != nil {
// 		return fmt.Errorf("error moving file %s to trash: %w", file, err)
// 	}

// 	fmt.Printf("File %s moved to trash at %s\n", file, dst_name)
// 	// Here you would add the actual logic to move the file to the trash
// 	return nil
// }
