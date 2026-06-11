package fingerprint

import (
	"encoding/binary"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSupports0107RawFingerprintData(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "far_0107", "raw_0107.bin")
	data := make([]byte, 4800)
	for i := 0; i < 24*100; i++ {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(i%4096))
	}
	writeTestFile(t, filePath, data)

	loader := NewLoader()
	if !loader.IsSupported(filePath) {
		t.Fatal("IsSupported() = false, want true")
	}

	img, format, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 24 || bounds.Dy() != 100 {
		t.Fatalf("image size = %dx%d, want 24x100", bounds.Dx(), bounds.Dy())
	}
	if format != "RAW/24x100" {
		t.Fatalf("format = %q, want RAW/24x100", format)
	}
}

func TestLoadSupports0307RawFingerprintData(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "fingerprint", "jv0307", "Samples", "raw", "0", "case_raw.bin")
	data := make([]byte, 43808)
	for i := 0; i < 148*148; i++ {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(i%4096))
	}
	writeTestFile(t, filePath, data)

	loader := NewLoader()
	if !loader.IsSupported(filePath) {
		t.Fatal("IsSupported() = false, want true")
	}

	img, format, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 148 || bounds.Dy() != 148 {
		t.Fatalf("image size = %dx%d, want 148x148", bounds.Dx(), bounds.Dy())
	}
	if format != "RAW/148x148" {
		t.Fatalf("format = %q, want RAW/148x148", format)
	}
}

func TestLoadSupports0307BinFingerprintData(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "fingerprint", "jv0307", "Samples", "bin", "0", "case_bin.bin")
	data := make([]byte, 102400)
	for i := 0; i < 160*160; i++ {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(i%65536))
	}
	writeTestFile(t, filePath, data)

	loader := NewLoader()
	if !loader.IsSupported(filePath) {
		t.Fatal("IsSupported() = false, want true")
	}

	img, format, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 160 || bounds.Dy() != 160 {
		t.Fatalf("image size = %dx%d, want 160x160", bounds.Dx(), bounds.Dy())
	}
	if format != "BIN/160x160" {
		t.Fatalf("format = %q, want BIN/160x160", format)
	}
}

func TestLoadSelectsAmbiguousRawProfileBySensorPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "frr_0106", "raw_0106.bin")
	data := make([]byte, 6240)
	for i := 0; i < 20*156; i++ {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(i%4096))
	}
	writeTestFile(t, filePath, data)

	img, format, err := NewLoader().Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 156 {
		t.Fatalf("image size = %dx%d, want 20x156", bounds.Dx(), bounds.Dy())
	}
	if format != "RAW/20x156" {
		t.Fatalf("format = %q, want RAW/20x156", format)
	}
}

func TestLoadUsesTenBitSensorProfile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "case_0201", "raw_0201.bin")
	data := make([]byte, 39192)
	binary.LittleEndian.PutUint16(data, 1023)
	writeTestFile(t, filePath, data)

	img, format, err := NewLoader().Load(filePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 138 || bounds.Dy() != 142 {
		t.Fatalf("image size = %dx%d, want 138x142", bounds.Dx(), bounds.Dy())
	}
	if format != "RAW/138x142" {
		t.Fatalf("format = %q, want RAW/138x142", format)
	}
	if got := img.At(0, 0); got != (color.Gray{Y: 255}) {
		t.Fatalf("pixel(0, 0) = %v, want 10-bit max scaled to 255", got)
	}
}

func writeTestFile(t *testing.T, filePath string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
