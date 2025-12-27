package storage

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/coffersTech/nanolog/server/internal/engine"
)

// NanoLog Header
var MagicHeader = []byte("NANOLOG1")

type ColumnWriter struct {
	encoder *zstd.Encoder
}

func NewColumnWriter() (*ColumnWriter, error) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	return &ColumnWriter{encoder: enc}, nil
}

// WriteSnapshot writes the MemTable to a .nano file.
func (cw *ColumnWriter) WriteSnapshot(filename string, mt *engine.MemTable) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// 1. Write Header
	if _, err := f.Write(MagicHeader); err != nil {
		return err
	}

	// 2. Prepare Data
	tsData := mt.TsCol
	lvlData := mt.LvlCol
	svcData := mt.SvcCol
	msgData := mt.MsgCol

	rowCount := uint32(len(tsData))
	if rowCount == 0 {
		// Even empty, write footer? Or just return.
		// Header + Footer.
		return cw.writeFooter(f, 0, 0, 0)
	}

	minTs := tsData[0]
	maxTs := tsData[rowCount-1]

	// 3. Compress and Write Columns

	// Timestamp (Int64)
	if err := cw.writeInt64Col(f, tsData); err != nil {
		return err
	}

	// Level (Uint8)
	if err := cw.writeUint8Col(f, lvlData); err != nil {
		return err
	}

	// Service (String)
	if err := cw.writeStringCol(f, svcData); err != nil {
		return err
	}

	// Message (String)
	if err := cw.writeStringCol(f, msgData); err != nil {
		return err
	}

	// 4. Footer
	return cw.writeFooter(f, rowCount, minTs, maxTs)
}

func (cw *ColumnWriter) writeInt64Col(f *os.File, data []int64) error {
	buf := new(bytes.Buffer)
	// Serialize: Just raw bytes
	for _, v := range data {
		binary.Write(buf, binary.LittleEndian, v)
	}
	return cw.compressAndWrite(f, buf.Bytes())
}

func (cw *ColumnWriter) writeUint8Col(f *os.File, data []uint8) error {
	buf := new(bytes.Buffer)
	// Serialize: Just raw bytes
	binary.Write(buf, binary.LittleEndian, data)
	return cw.compressAndWrite(f, buf.Bytes())
}

func (cw *ColumnWriter) writeStringCol(f *os.File, data []string) error {
	buf := new(bytes.Buffer)
	// Serialize: [Len uint32][Bytes]...
	// We don't write count here because Header/Footer has RowCount,
	// but for safety we could.
	// Let's just write [Len][Content].
	for _, s := range data {
		b := []byte(s)
		binary.Write(buf, binary.LittleEndian, uint32(len(b)))
		buf.Write(b)
	}
	return cw.compressAndWrite(f, buf.Bytes())
}

func (cw *ColumnWriter) compressAndWrite(f *os.File, raw []byte) error {
	compressed := cw.encoder.EncodeAll(raw, make([]byte, 0, len(raw)))

	// Write Compressed Size (uint32)
	size := uint32(len(compressed))
	if err := binary.Write(f, binary.LittleEndian, size); err != nil {
		return err
	}

	// Write Data
	_, err := f.Write(compressed)
	return err
}

func (cw *ColumnWriter) writeFooter(f *os.File, rowCount uint32, minTs, maxTs int64) error {
	// RowCount (4) + MinTs (8) + MaxTs (8)
	if err := binary.Write(f, binary.LittleEndian, rowCount); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, minTs); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, maxTs); err != nil {
		return err
	}
	return nil
}
