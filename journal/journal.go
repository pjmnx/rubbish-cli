package journal

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
)

type Journal struct {
	Path string
	db   *badger.DB
}

func (j *Journal) Load() error {
	var err error

	if j.Path == "" {
		return fmt.Errorf("journal path is not set")
	}
	j.Path = filepath.Clean(j.Path)

	if j.db == nil {
		j.db, err = badger.Open(badger.DefaultOptions(j.Path))

		if err != nil {
			return fmt.Errorf("error opening badger database: %w", err)
		}
	}

	return nil
}

func (m *MetaData) marshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (j *Journal) Close() error {
	if j.db != nil {
		return j.db.Close()
	}
	return nil
}

func (j *Journal) Add(item string, file string, swipeTime int) error {
	path, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("error getting absolute path for file %s: %w", file, err)
	}
	metadata := GenerateMetadata(item, path, swipeTime)
	if metadata == nil {
		return fmt.Errorf("error generating metadata for item: %s", item)
	}

	return j.register(metadata)
}

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

func (j *Journal) Delete(item string) error {
	if j.db == nil {
		return fmt.Errorf("journal database is not initialized")
	}

	return j.db.Update(func(txn *badger.Txn) error {
		key := []byte(item)
		return txn.Delete(key)
	})
}

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
