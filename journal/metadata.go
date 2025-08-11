package journal

import (
	"os"
	"time"
)

// MetaData represents the metadata information for an item that has been moved to trash.
// It contains all necessary information to track, restore, and manage the lifecycle
// of trashed files and directories.
type MetaData struct {
	// Item is the unique identifier for the trashed item, typically the basename
	// of the file or directory that was moved to trash
	Item string

	// Origin is the absolute path of the original file or directory before it was
	// moved to trash, used for restoration purposes
	Origin string

	// Type indicates the kind of filesystem object (file, directory, symlink, etc.)
	// using the defined type constants
	Type uint

	// WipeoutTime is the number of days the item should remain in trash before
	// it becomes eligible for permanent deletion
	WipeoutTime int

	// TossedTime is the Unix timestamp (seconds since epoch) when the item
	// was originally moved to trash
	TossedTime int64
}

// File system type constants for categorizing trashed items.
// These constants are used to identify the type of filesystem object
// that was moved to trash, which affects how restoration and cleanup
// operations are performed.
const (
	// TypeFile represents a regular file
	TypeFile = iota + 1

	// TypeDirectory represents a directory/folder
	TypeDirectory

	// TypeSymlink represents a symbolic link
	TypeSymlink

	// TypeOther represents any other type of filesystem object
	// (special files, devices, etc.)
	TypeOther
)

// getType determines the filesystem type of the item at the given path.
// It uses os.Lstat to examine the file without following symbolic links,
// allowing proper identification of symlinks themselves.
//
// The function categorizes filesystem objects into one of four types:
// - TypeSymlink: for symbolic links
// - TypeDirectory: for directories/folders
// - TypeFile: for regular files
// - TypeOther: for special files, devices, or when an error occurs
//
// Parameters:
//   - path: The filesystem path to examine
//
// Returns the appropriate type constant (TypeFile, TypeDirectory, TypeSymlink, or TypeOther).
// If an error occurs during file examination, TypeOther is returned.
func getType(path string) uint {
	info, err := os.Lstat(path)
	if err != nil {
		return TypeOther
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return TypeSymlink
	}
	if info.IsDir() {
		return TypeDirectory
	}
	return TypeFile
}

// GenerateMetadata creates a new MetaData struct with the provided information
// and automatically fills in the current timestamp and filesystem type.
// This function is the primary way to create metadata entries for items
// being moved to trash.
//
// The function automatically:
// - Sets the current Unix timestamp as the TossedTime
// - Determines the filesystem type by examining the item path
// - Initializes all fields with the provided values
//
// Parameters:
//   - item: Unique identifier for the trashed item (typically the filename)
//   - path: Absolute path to the original location of the file/directory
//   - wipeoutTime: Number of days the item should remain in trash before cleanup
//
// Returns a pointer to a newly created MetaData struct with all fields populated.
// Note: The Type field is determined by examining the 'item' parameter, not the 'path'.
func GenerateMetadata(item string, path string, wipeoutTime int) *MetaData {
	return &MetaData{
		Item:        item,
		Origin:      path,
		Type:        getType(item),
		WipeoutTime: wipeoutTime,
		TossedTime:  time.Now().Unix(),
	}
}

func (m *MetaData) TossElapsed() time.Duration {
	// Calculate the elapsed time since the item was tossed to trash
	return time.Since(time.Unix(m.TossedTime, 0))
}

func (m *MetaData) IsWipeable() bool {
	// Check if the item is eligible for wipeout based on its WipeoutTime
	return m.TossElapsed().Hours()/24.0 >= float64(m.WipeoutTime)
}
