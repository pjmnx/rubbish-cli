package journal

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
)

// Journal represents a persistent storage system for tracking metadata
// of files that have been moved to trash. It uses BadgerDB as the underlying
// storage engine to maintain a record of all trash operations.
type Journal struct {
	Path string     // Path to the directory where the journal database is stored
	db   *badger.DB // BadgerDB instance for persistent storage
}

// Load initializes the journal database at the specified path.
// It opens a BadgerDB instance and prepares it for operations.
// Returns an error if the path is not set or if the database cannot be opened.
func (j *Journal) Load() error {
	var err error

	if j.Path == "" {
		return fmt.Errorf("journal path is not set")
	}
	j.Path = filepath.Clean(j.Path)

	if j.db == nil {

		j.db, err = badger.Open(badger.DefaultOptions(j.Path).WithLoggingLevel(badger.ERROR))

		if err != nil {
			return fmt.Errorf("error opening badger database: %w", err)
		}
	}

	return nil
}

// marshalBinary converts a MetaData struct to JSON bytes for storage.
// This method implements binary marshaling for the MetaData type,
// allowing it to be stored efficiently in the BadgerDB database.
// Returns the JSON representation as bytes or an error if marshaling fails.
func (m *MetaData) marshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

// Close safely closes the journal database connection.
// This should be called when the journal is no longer needed to ensure
// proper cleanup of database resources and prevent data corruption.
// Returns an error if the database close operation fails.
func (j *Journal) Close() error {
	if j.db != nil {
		j.db.Sync()
		return j.db.Close()
	}
	return nil
}

// Add creates a new journal entry for a file that has been moved to trash.
// It generates metadata for the specified item and file path, then stores
// it in the journal database for tracking purposes.
//
// Parameters:
//   - item: A unique identifier for the trashed item
//   - file: The original file path before it was moved to trash
//   - wipeoutTime: The number of days before the item can be permanently deleted
//
// Returns an error if the absolute path cannot be determined, metadata generation
// fails, or the database operation encounters an issue.
func (j *Journal) Add(item string, file string, wipeoutTime int) error {
	path, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("error getting absolute path for file %s: %w", file, err)
	}
	metadata := GenerateMetadata(item, path, wipeoutTime)
	if metadata == nil {
		return fmt.Errorf("error generating metadata for item: %s", item)
	}

	return j.register(metadata)
}

// register stores metadata in the journal database using a database transaction.
// This is an internal method that handles the low-level database operations
// for storing metadata entries. It ensures atomic writes to the database.
//
// Parameters:
//   - metadata: The MetaData struct containing information about the trashed item
//
// Returns an error if the database is not initialized or if the database
// transaction fails during the write operation.
func (j *Journal) register(metadata *MetaData) error {
	if j.db == nil {
		return fmt.Errorf("journal database is not initialized")
	}

	return j.db.Update(func(txn *badger.Txn) error {
		key := []byte(metadata.Item)
		value, err := metadata.marshalBinary()
		if err != nil {
			return fmt.Errorf("error marshaling metadata: %w", err)
		}
		return txn.Set(key, value)
	})
}

// Get retrieves metadata for a specific item from the journal database.
// This method looks up an item by its unique identifier and returns
// the associated metadata containing information about when it was trashed,
// its original location, and retention settings.
//
// Parameters:
//   - item: The unique identifier of the item to retrieve
//
// Returns the MetaData struct for the item, or an error if the database
// is not initialized, the item is not found, or unmarshaling fails.
func (j *Journal) Get(item string) (*MetaData, error) {
	if j.db == nil {
		return nil, fmt.Errorf("journal database is not initialized")
	}

	var metadata MetaData
	err := j.db.View(func(txn *badger.Txn) error {
		key := []byte(item)
		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("error getting metadata: %w", err)
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &metadata)
		})
	})

	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

// List retrieves all metadata entries from the journal database.
// This method iterates through all stored items and returns a slice
// containing metadata for every item currently in the trash.
// Useful for displaying trash contents and generating status reports.
//
// Returns a slice of MetaData pointers for all items in the journal,
// or an error if the database is not initialized or if any unmarshaling
// operation fails during iteration.
func (j *Journal) List() ([]*MetaData, error) {
	if j.db == nil {
		return nil, fmt.Errorf("journal database is not initialized")
	}

	var metadataList []*MetaData
	err := j.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var metadata MetaData
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &metadata)
			})
			if err != nil {
				return fmt.Errorf("error unmarshaling metadata: %w", err)
			}
			metadataList = append(metadataList, &metadata)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return metadataList, nil
}

// Delete removes a specific item's metadata from the journal database.
// This method is typically called when an item is either restored from
// trash or permanently deleted after its retention period expires.
//
// Parameters:
//   - item: The unique identifier of the item to remove from the journal
//
// Returns an error if the database is not initialized or if the
// deletion operation fails.
func (j *Journal) Delete(item string) error {
	if j.db == nil {
		return fmt.Errorf("journal database is not initialized")
	}

	return j.db.Update(func(txn *badger.Txn) error {
		key := []byte(item)
		return txn.Delete(key)
	})
}

// Clear removes all entries from the journal database.
// This method performs a complete cleanup of the journal, removing
// metadata for all items. Use with caution as this operation cannot
// be undone and will result in loss of all trash tracking information.
//
// Returns an error if the database is not initialized or if any
// deletion operation fails during the clearing process.
func (j *Journal) Clear() error {
	if j.db == nil {
		return fmt.Errorf("journal database is not initialized")
	}

	return j.db.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			if err := txn.Delete(item.Key()); err != nil {
				return fmt.Errorf("error deleting metadata: %w", err)
			}
		}
		return nil
	})
}

// Count returns the total number of items currently tracked in the journal.
// This method provides a quick way to determine how many items are
// currently in the trash without retrieving all the metadata.
//
// Returns the count of items in the journal database, or an error
// if the database is not initialized or if the counting operation fails.
func (j *Journal) Count() (int, error) {
	if j.db == nil {
		return 0, fmt.Errorf("journal database is not initialized")
	}

	count := 0
	err := j.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})

	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetSize calculates the total size of all metadata stored in the journal database.
// This method iterates through all entries and sums up the size of their
// stored values, providing insight into the storage overhead of the journal.
//
// Returns the total size in bytes of all metadata entries, or an error
// if the database is not initialized or if the size calculation fails.
func (j *Journal) GetSize() (int64, error) {
	if j.db == nil {
		return 0, fmt.Errorf("journal database is not initialized")
	}

	var size int64
	err := j.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			size += item.ValueSize()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}
	return size, nil
}

func (j *Journal) GetContainerItems(container string) ([]*MetaData, error) {
	if j.db == nil {
		return nil, fmt.Errorf("journal database is not initialized")
	}

	metadataList, err := j.GetAllItems()
	if err != nil {
		return nil, fmt.Errorf("error retrieving all items: %w", err)
	}

	var containerItems []*MetaData
	for _, metadata := range metadataList {
		if strings.HasPrefix(metadata.Origin, container) {
			containerItems = append(containerItems, metadata)
		}
	}

	if len(containerItems) == 0 {
		return nil, nil
	}
	return containerItems, nil
}

func (j *Journal) GetAllItems() ([]*MetaData, error) {
	if j.db == nil {
		return nil, fmt.Errorf("journal database is not initialized")
	}
	var metadataList []*MetaData
	// Retrieve all items from the journal database
	// This method iterates through all stored items and returns a slice
	// containing metadata for every item currently in the trash.
	// Useful for displaying trash contents and generating status reports.
	// Returns a slice of MetaData pointers for all items in the journal,
	// or an error if the database is not initialized or if any unmarshaling
	// operation fails during iteration.

	err := j.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var metadata MetaData
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &metadata)
			})
			if err != nil {
				return fmt.Errorf("error unmarshaling metadata: %w", err)
			}
			metadataList = append(metadataList, &metadata)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return metadataList, nil
}
