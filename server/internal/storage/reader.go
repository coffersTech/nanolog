package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/coffersTech/nanolog/server/internal/engine"
	"github.com/klauspost/compress/zstd"
)

var ErrInvalidHeader = errors.New("invalid .nano file header")

// LogIterator provides a row-by-row view of logs.
type LogIterator interface {
	Next() bool
	Row() engine.LogRow
	Error() error
	Close() error
}

type ColumnReader struct {
	decoder *zstd.Decoder
}

func NewColumnReader() (*ColumnReader, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	return &ColumnReader{decoder: dec}, nil
}

// NewIterator creates a new iterator for a .nano file with filtering.
func (cr *ColumnReader) NewIterator(filename string, filter engine.Filter) (LogIterator, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	it := &FileIterator{
		reader: cr,
		file:   f,
		filter: filter,
	}

	if err := it.init(); err != nil {
		f.Close()
		return nil, err
	}

	return it, nil
}

type FileIterator struct {
	reader *ColumnReader
	file   *os.File
	filter engine.Filter

	// Columns data
	timestamps []int64
	levels     []uint8
	services   []string
	messages   []string

	rowCount int
	cursor   int
	currRow  engine.LogRow
	err      error
}

func (it *FileIterator) init() error {
	// 1. Validate Header
	header := make([]byte, 8)
	if _, err := io.ReadFull(it.file, header); err != nil {
		return err
	}
	if !bytes.Equal(header, MagicHeader) {
		return ErrInvalidHeader
	}

	// 2. Read Footer (at end of file)
	// Footer: RowCount(4) + MinTs(8) + MaxTs(8) = 20 bytes
	info, err := it.file.Stat()
	if err != nil {
		return err
	}
	if info.Size() < 28 { // Header(8) + Footer(20)
		return errors.New("file too small")
	}

	footer := make([]byte, 20)
	if _, err := it.file.ReadAt(footer, info.Size()-20); err != nil {
		return err
	}

	rowCount := binary.LittleEndian.Uint32(footer[0:4])
	minTs := int64(binary.LittleEndian.Uint64(footer[4:12]))
	maxTs := int64(binary.LittleEndian.Uint64(footer[12:20]))

	it.rowCount = int(rowCount)
	it.cursor = -1

	// File-level filtering based on MinTs/MaxTs
	if rowCount > 0 {
		if it.filter.MinTime > 0 && maxTs < it.filter.MinTime {
			it.rowCount = 0 // Skip entire file
			return nil
		}
		if it.filter.MaxTime > 0 && minTs > it.filter.MaxTime {
			it.rowCount = 0 // Skip entire file
			return nil
		}
	}

	// 3. Read and decompress all columns (in-memory for now per block)
	// Note: True streaming would decompress on demand, but .nano v1 stores
	// whole columns as single compressed blocks.
	// We still benefit from row-by-row processing at the engine level.

	tsData, err := it.reader.readAndDecompress(it.file)
	if err != nil {
		return err
	}
	it.timestamps = bytesToInt64Slice(tsData)

	lvlData, err := it.reader.readAndDecompress(it.file)
	if err != nil {
		return err
	}
	it.levels = lvlData

	svcData, err := it.reader.readAndDecompress(it.file)
	if err != nil {
		return err
	}
	it.services = bytesToStringSlice(svcData)

	msgData, err := it.reader.readAndDecompress(it.file)
	if err != nil {
		return err
	}
	it.messages = bytesToStringSlice(msgData)

	// Basic column length validation
	if it.rowCount != len(it.levels) || it.rowCount != len(it.services) || it.rowCount != len(it.messages) {
		return errors.New("column length mismatch")
	}

	return nil
}

func (it *FileIterator) Next() bool {
	for {
		it.cursor++
		if it.cursor >= it.rowCount {
			return false
		}

		// Apply filters
		ts := it.timestamps[it.cursor]
		if it.filter.MinTime > 0 && ts < it.filter.MinTime {
			continue
		}
		if it.filter.MaxTime > 0 && ts > it.filter.MaxTime {
			continue
		}

		lvl := it.levels[it.cursor]
		if it.filter.Level > 0 && lvl != it.filter.Level {
			continue
		}

		svc := it.services[it.cursor]
		if it.filter.Service != "" && svc != it.filter.Service {
			continue
		}

		msg := it.messages[it.cursor]
		if it.filter.Query != "" && !strings.Contains(msg, it.filter.Query) {
			continue
		}

		// Match found
		it.currRow = engine.LogRow{
			Timestamp: ts,
			Level:     lvl,
			Service:   svc,
			Message:   msg,
		}
		return true
	}
}

func (it *FileIterator) Row() engine.LogRow {
	return it.currRow
}

func (it *FileIterator) Error() error {
	return it.err
}

func (it *FileIterator) Close() error {
	return it.file.Close()
}

// ReadSnapshot reads a .nano file and returns log rows matching the filter.
func (cr *ColumnReader) ReadSnapshot(filename string, filter engine.Filter) ([]engine.LogRow, error) {
	it, err := cr.NewIterator(filename, filter)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var rows []engine.LogRow
	for it.Next() {
		rows = append(rows, it.Row())
	}
	return rows, it.Error()
}

// readAndDecompress reads a compressed block (size + data) and decompresses it.
func (cr *ColumnReader) readAndDecompress(r io.Reader) ([]byte, error) {
	// Read compressed size (uint32)
	var size uint32
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	// Read compressed data
	compressed := make([]byte, size)
	if _, err := io.ReadFull(r, compressed); err != nil {
		return nil, err
	}

	// Decompress
	decompressed, err := cr.decoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

// bytesToInt64Slice converts a byte slice to []int64 (LittleEndian).
func bytesToInt64Slice(data []byte) []int64 {
	count := len(data) / 8
	result := make([]int64, count)
	buf := bytes.NewReader(data)
	for i := 0; i < count; i++ {
		binary.Read(buf, binary.LittleEndian, &result[i])
	}
	return result
}

// bytesToStringSlice converts a byte slice to []string.
// Format: [Len uint32][Bytes]...
func bytesToStringSlice(data []byte) []string {
	var result []string
	buf := bytes.NewReader(data)

	for buf.Len() > 0 {
		var length uint32
		if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
			break
		}
		strBytes := make([]byte, length)
		if _, err := io.ReadFull(buf, strBytes); err != nil {
			break
		}
		result = append(result, string(strBytes))
	}

	return result
}
