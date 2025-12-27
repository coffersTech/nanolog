package engine

import (
	"fmt"
	"path/filepath"
)

// FlushFunc is a function type that writes a MemTable to a file.
// This allows the engine package to not depend on storage package directly.
type FlushFunc func(filename string, mt *MemTable) error

// FlushMemTable flushes the MemTable to disk using the provided writer function.
// Filename format: log_{MinTimestamp}_{MaxTimestamp}.nano
func FlushMemTable(mt *MemTable, dataDir string, writerFn FlushFunc) error {
	if mt.Len() == 0 {
		return nil
	}

	minTs := mt.MinTimestamp()
	maxTs := mt.MaxTimestamp()
	filename := fmt.Sprintf("log_%d_%d.nano", minTs, maxTs)
	path := filepath.Join(dataDir, filename)

	if err := writerFn(path, mt); err != nil {
		return err
	}

	mt.Reset()
	return nil
}
