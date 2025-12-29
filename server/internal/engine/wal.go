package engine

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// WAL handles write-ahead logging to prevent data loss during crashes.
type WAL struct {
	file *os.File
	path string
	mu   sync.Mutex
}

// OpenWAL opens or creates a WAL file at the specified path.
func OpenWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{
		file: f,
		path: path,
	}, nil
}

// Write records a log row to the WAL.
func (w *WAL) Write(ts int64, level, service, host, msg string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	row := LogRow{
		Timestamp: ts,
		Level:     EncodeLevel(level),
		Service:   service,
		Host:      host,
		Message:   msg,
	}

	data, err := json.Marshal(row)
	if err != nil {
		return err
	}

	// Format: [Len uint32][JSON Bytes]
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))

	if _, err := w.file.Write(lenBuf); err != nil {
		return err
	}
	if _, err := w.file.Write(data); err != nil {
		return err
	}

	return nil
}

// Sync flushes the WAL file buffers to disk.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

// Reset truncates the WAL file.
func (w *WAL) Reset() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Truncate(0); err != nil {
		return err
	}
	_, err := w.file.Seek(0, 0)
	return err
}

// Close closes the WAL file.
func (w *WAL) Close() error {
	return w.file.Close()
}

// Replay reads the WAL and returns all log rows.
func (w *WAL) Replay() ([]LogRow, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, 0); err != nil {
		return nil, err
	}

	var rows []LogRow
	for {
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(w.file, lenBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, fmt.Errorf("WAL replay error (len): %v", err)
		}

		length := binary.LittleEndian.Uint32(lenBuf)
		data := make([]byte, length)
		if _, err := io.ReadFull(w.file, data); err != nil {
			return rows, fmt.Errorf("WAL replay error (data): %v", err)
		}

		var row LogRow
		if err := json.Unmarshal(data, &row); err != nil {
			return rows, fmt.Errorf("WAL replay error (unmarshal): %v", err)
		}
		rows = append(rows, row)
	}

	return rows, nil
}
