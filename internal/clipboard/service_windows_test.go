package clipboard

import (
	"encoding/binary"
	"testing"
)

func TestCalcHashIsStableForSameFiles(t *testing.T) {
	files := []string{`C:\temp\a.bmp`, `C:\temp\b.bmp`}

	first := calcHash(files)
	second := calcHash([]string{`C:/TEMP/a.bmp`, `C:/TEMP/b.bmp`})

	if first != second {
		t.Fatal("calcHash() should stay stable for the same Windows file list")
	}
}

func TestParseCIDAOffsets(t *testing.T) {
	data := make([]byte, 64)
	binary.LittleEndian.PutUint32(data[0:], 2)
	binary.LittleEndian.PutUint32(data[4:], 16)
	binary.LittleEndian.PutUint32(data[8:], 28)
	binary.LittleEndian.PutUint32(data[12:], 44)

	offsets, err := parseCIDAOffsets(data)
	if err != nil {
		t.Fatalf("parseCIDAOffsets() error = %v", err)
	}

	want := []uint32{16, 28, 44}
	if len(offsets) != len(want) {
		t.Fatalf("parseCIDAOffsets() len = %d, want %d", len(offsets), len(want))
	}
	for i := range want {
		if offsets[i] != want[i] {
			t.Fatalf("parseCIDAOffsets()[%d] = %d, want %d", i, offsets[i], want[i])
		}
	}
}

func TestParseCIDAOffsetsRejectsTruncatedTable(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[0:], 2)

	if _, err := parseCIDAOffsets(data); err == nil {
		t.Fatal("parseCIDAOffsets() should reject truncated offset tables")
	}
}

func TestParseCIDAOffsetsRejectsOutOfRangeOffset(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:], 1)
	binary.LittleEndian.PutUint32(data[4:], 12)
	binary.LittleEndian.PutUint32(data[8:], 128)

	if _, err := parseCIDAOffsets(data); err == nil {
		t.Fatal("parseCIDAOffsets() should reject out-of-range offsets")
	}
}
