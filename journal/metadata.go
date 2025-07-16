package journal

import (
	"os"
	"time"
)

type MetaData struct {
	// Add fields as necessary to store metadata about the rubbish
	Item       string
	Abs        string // The original file path
	Type       uint
	SwipeTime  int
	TossedTime int64 // Unix timestamp when the item was tossed
}

const (
	TypeFile = iota + 1
	TypeDirectory
	TypeSymlink
	TypeOther
)

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

func GenerateMetadata(item string, path string, swipeTime int) *MetaData {
	return &MetaData{
		Item:       item,
		Abs:        path,
		Type:       getType(item),
		SwipeTime:  swipeTime,
		TossedTime: time.Now().Unix(),
	}
}
