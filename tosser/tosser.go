package tosser

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"rubbish/config"
	"slices"
	"syscall"
	"time"
)

var (
	Flags             = flag.NewFlagSet("toss", flag.ExitOnError)
	retentionTime int = -1
	silentMode    bool
)

func init() {
	Flags.IntVar(&retentionTime, "r", -1, "Time to retain the file before it is wiped out from the filesystem.")
	Flags.BoolVar(&silentMode, "s", false, "Silent mode. Suppress non-error messages.")

	Flags.Usage = func() {
		fmt.Println("Toss moves the specified files to the rubbish bin.\n\n",
			"Usage:\n\n",
			"\trubbish toss [options] <file1> <file2> ...\n\n",
			"Options:")
		Flags.PrintDefaults()
	}
}

// Command handles the "toss" command which moves files and directories to trash.
// It processes command-line arguments to determine retention time and validates
// each file before moving it to the trash container. The function supports
// custom retention periods via the --retention flag.
//
// The command performs the following operations:
// 1. Parses command-line flags for retention time override
// 2. Validates that files/directories exist and are accessible
// 3. Determines whether each item is a file or directory
// 4. Delegates to appropriate tossing functions
// 5. Updates the journal with metadata about tossed items
// 6. Provides feedback about the operation results
//
// Parameters:
//   - args: Command-line arguments passed to the toss command
//   - cfg: Application configuration containing default settings and journal
//
// Returns an error if no files are specified, if any file cannot be accessed,
// or if the tossing operation fails for any item.
func Command(args []string, cfg *config.Config) error {
	if len(args) == 0 {
		return fmt.Errorf("no files or directory specified to toss")
	}

	if retentionTime >= 0 {
		cfg.WipeoutTime = retentionTime
	}

	for _, file := range args {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("invalid rubbish to toss '%s': %w", file, err)
		}

		if err := Toss(file, cfg); err != nil {
			return fmt.Errorf("error tossing rubbish %s: %w", file, err)
		}

		if silentMode {
			continue
		}

		fmt.Printf("\033[32mTossed\033[0m '%s' to rubbish bin. ", file)
		if cfg.WipeoutTime == 0 {
			fmt.Println("Wipeout immediate.")
		} else {
			fmt.Printf("Wipeout after %d days.\n", cfg.WipeoutTime)
		}
	}

	if silentMode {
		return nil
	}

	if size, err := config.BinSize(cfg); err != nil {
		fmt.Printf("Error determining rubbish bin size: %v\n", err)
	} else {
		fmt.Printf("Bin size: %s\n", config.ReadableSize(uint64(size)))
	}

	return nil
}

func NameSufix(size uint) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// func userHasOwnership(stat *syscall.Stat_t) bool {
// 	currentUID := os.Getuid()
// 	currentGID := os.Getgid()
// 	return stat.Uid == uint32(currentUID) || stat.Gid == uint32(currentGID)
// }

// func validateAccessOld(item string) error {
// 	info, err := os.Stat(item)
// 	if err != nil {
// 		return fmt.Errorf("cannot access item %s: %w", item, err)
// 	}

// 	// Check write permission based on file mode
// 	if info.Mode().Perm()&0200 == 0 {
// 		return fmt.Errorf("no write permission for item %s", item)
// 	}

// 	// Get file system statistics
// 	stat, ok := info.Sys().(*syscall.Stat_t)
// 	if !ok {
// 		return fmt.Errorf("cannot get detailed file information for %s", item)
// 	}

// 	// Check user and group ownership
// 	if !userHasOwnership(stat) {
// 		return fmt.Errorf("you don't have ownership of %s", item)
// 	}

// 	return nil
// }

func validateParentDirAccess(item string) error {
	parentDir := filepath.Dir(item)
	uid := os.Getuid()
	gid := os.Getgid()

	info, err := os.Stat(parentDir)
	if err != nil {
		return fmt.Errorf("cannot access parent directory of %s: %w", item, err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to get parent directory system info")
	}

	parentUID := int(stat.Uid)
	parentGID := int(stat.Gid)
	parentMode := info.Mode()

	// Check parent directory write permission
	if uid == parentUID && parentMode&0200 != 0 {
		return nil
	}

	inGroup := gid == parentGID
	{
		suppGroups, err := os.Getgroups()
		if err != nil {
			return fmt.Errorf("failed to get supplementary groups: %w", err)
		}
		if slices.Contains(suppGroups, parentGID) {
			inGroup = true
		}
	}

	if inGroup && parentMode&0020 != 0 {
		return nil
	}

	if parentMode&0002 != 0 {
		return nil
	}

	return fmt.Errorf("no write permission for parent directory of %s", item)
}

func validateAccess(item string) error {
	// Getting the current user ID and group ID
	uid := os.Getuid()
	gid := os.Getgid()

	if uid == 0 {
		// Root user has access to everything
		return nil
	}

	// Get file wonership and permissions
	info, err := os.Stat(item)
	if err != nil {
		return fmt.Errorf("cannot access item %s: %w", item, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot get detailed file information for %s", item)
	}
	fileUid := int(stat.Uid)
	fileGid := int(stat.Gid)
	fileMode := info.Mode().Perm()

	// Check if user is the owner
	if uid == fileUid {
		// Owner needs write permission
		if fileMode&0200 != 0 {
			return validateParentDirAccess(item)
		}
		return fmt.Errorf("no write permission for item %s", item)
	}

	// Check if user is in the same group
	inGroup := fileGid == gid

	if !inGroup {
		suppGroups, err := os.Getgroups()
		if err != nil {
			return fmt.Errorf("failed to get supplementary groups: %w", err)
		}
		if slices.Contains(suppGroups, fileGid) {
			inGroup = true
		}
	}

	if inGroup {
		// Group member needs group write permission
		if fileMode&0020 != 0 {
			return validateParentDirAccess(item)
		}
		return fmt.Errorf("group does not have write permission")
	}

	// Check others permission
	if fileMode&0002 != 0 {
		return validateParentDirAccess(item)
	}

	return fmt.Errorf("user is not owner and not in group")
}

func Toss(item string, cfg *config.Config) error {
	if err := validateAccess(item); err != nil {
		return err
	}

	destination := path.Join(cfg.ContainerPath, filepath.Base(item+"_"+NameSufix(6)))

	if err := cfg.Journal.AddFileByName(filepath.Base(destination), item, cfg.WipeoutTime); err != nil {
		return fmt.Errorf("error adding item to rubbish journal: %v", err)
	}

	if err := os.Rename(item, destination); err != nil {
		if errj := cfg.Journal.Delete(filepath.Base(destination)); errj != nil {
			return fmt.Errorf("error deleting journal entry for %s due to unable to move to rubbish bin: %w", item, errj)
		}
		return fmt.Errorf("error moving item to rubbish bin: %v", err)
	}

	return nil
}
