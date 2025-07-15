package tosser

import (
	"flag"
	"fmt"
	"os"
	"path"
	"rubbish/config"
	"time"
)

func Command(args []string, cfg *config.Config) error {
	fs := flag.NewFlagSet("toss", flag.ExitOnError)

	local_cfg := cfg

	fs.IntVar(&local_cfg.SwipeTime, "wait", local_cfg.SwipeTime, "Time to wait before swiping the file(s) from the trash (in days)")

	fs.Parse(args)

	if len(fs.Args()) == 0 {
		return fmt.Errorf("no files or directory specified to toss")
	}

	for _, file := range fs.Args() {
		if file == "" {
			return fmt.Errorf("no files specified to toss")
		}

		err := TossFile(file, cfg)
		if err != nil {
			return fmt.Errorf("error tossing file %s: %w", file, err)
		}
	}

	// fmt.Println("Moving files to trash:", fs.Args())
	fmt.Fprintf(os.Stdout, "\033[32mTossing\033[0m files %s to trash with a wait time of %d days.\n", fs.Args(), local_cfg.SwipeTime)
	return nil
}

func TossFile(file string, cfg *config.Config) error {
	// Implement the logic to move the file to the trash
	// This is a placeholder implementation

	// wd, err := os.Getwd()
	// if err != nil {
	// 	return fmt.Errorf("error getting current working directory: %w", err)
	// }

	// file_data := config.GenerateMetadata(file, os.Getuid(), wd, cfg.ContainerPath, cfg.SwipeTime)
	fmt.Printf("Tossing file: %s with wait time: %d days\n", file, cfg.SwipeTime)

	dst_name := cfg.ContainerPath + "/" + file

	if _, err := os.Stat(dst_name); err == nil {
		// return fmt.Errorf("file %s already exists in the trash", dst_name)
		dst_name = path.Join(cfg.ContainerPath, file+"_"+time.Now().Format("20060102150405"))
		fmt.Printf("File already exists in the trash, renaming to: %s\n", dst_name)
	}

	err := os.Rename(file, dst_name) // Move file to the trash directory
	if err != nil {
		return fmt.Errorf("error moving file %s to trash: %w", file, err)
	}

	fmt.Printf("File %s moved to trash at %s\n", file, dst_name)
	// Here you would add the actual logic to move the file to the trash
	return nil
}
