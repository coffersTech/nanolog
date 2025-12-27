package engine

import (
	"sync"
)

// ColumnType defines the type of data stored in a column.
type ColumnType int

const (
	ColumnTypeInt64 ColumnType = iota
	ColumnTypeInt8
	ColumnTypeBytes
)

// Column is the generic interface for a column in the MemTable.
type Column interface {
	Type() ColumnType
	Reset()
	Size() int  // Number of generic rows/elements
	Bytes() int // Estimated memory usage in bytes
}

// Int64Column stores int64 values (e.g., Timestamp).
type Int64Column struct {
	Data []int64
}

func NewInt64Column(capacity int) *Int64Column {
	return &Int64Column{
		Data: make([]int64, 0, capacity),
	}
}

func (c *Int64Column) Type() ColumnType {
	return ColumnTypeInt64
}

func (c *Int64Column) Append(v int64) {
	c.Data = append(c.Data, v)
}

func (c *Int64Column) Reset() {
	c.Data = c.Data[:0]
}

func (c *Int64Column) Size() int {
	return len(c.Data)
}

func (c *Int64Column) Bytes() int {
	return len(c.Data) * 8
}

// Int8Column stores int8 values (e.g., Level).
type Int8Column struct {
	Data []int8
}

func NewInt8Column(capacity int) *Int8Column {
	return &Int8Column{
		Data: make([]int8, 0, capacity),
	}
}

func (c *Int8Column) Type() ColumnType {
	return ColumnTypeInt8
}

func (c *Int8Column) Append(v int8) {
	c.Data = append(c.Data, v)
}

func (c *Int8Column) Reset() {
	c.Data = c.Data[:0]
}

func (c *Int8Column) Size() int {
	return len(c.Data)
}

func (c *Int8Column) Bytes() int {
	return len(c.Data)
}

// BytesColumn stores variable-length byte slices using a flat buffer and offsets.
// This reduces GC pressure compared to []string or [][]byte.
type BytesColumn struct {
	Data    []byte // The flat buffer storing all bytes
	Offsets []int  // Starting offset for each row. Length is RowCount + 1
}

func NewBytesColumn(dataCap, rowsCap int) *BytesColumn {
	c := &BytesColumn{
		Data:    make([]byte, 0, dataCap),
		Offsets: make([]int, 0, rowsCap+1),
	}
	c.Offsets = append(c.Offsets, 0) // Initial offset
	return c
}

func (c *BytesColumn) Type() ColumnType {
	return ColumnTypeBytes
}

// Append adds a byte slice to the column.
// Safe for zero-allocation if input is transient, as we copy data.
func (c *BytesColumn) Append(v []byte) {
	c.Data = append(c.Data, v...)
	c.Offsets = append(c.Offsets, len(c.Data))
}

// AppendString adds a string to the column.
func (c *BytesColumn) AppendString(v string) {
	c.Data = append(c.Data, v...)
	c.Offsets = append(c.Offsets, len(c.Data))
}

func (c *BytesColumn) Reset() {
	c.Data = c.Data[:0]
	c.Offsets = c.Offsets[:0]
	c.Offsets = append(c.Offsets, 0)
}

func (c *BytesColumn) Size() int {
	return len(c.Offsets) - 1
}

func (c *BytesColumn) Bytes() int {
	return len(c.Data) + len(c.Offsets)*8
}

// Get returns the byte slice at index i.
// The returned slice is valid until the next Reset or potentially re-allocation trigger (though append only grows).
// To be safe, treat as read-only or copy if needed outside scope.
func (c *BytesColumn) Get(i int) []byte {
	if i < 0 || i >= len(c.Offsets)-1 {
		return nil
	}
	start := c.Offsets[i]
	end := c.Offsets[i+1]
	return c.Data[start:end]
}

// Pool for columns to reuse memory
var (
	int64ColPool = sync.Pool{
		New: func() interface{} { return NewInt64Column(4096) },
	}
	bytesColPool = sync.Pool{
		New: func() interface{} { return NewBytesColumn(64*1024, 4096) },
	}
)
