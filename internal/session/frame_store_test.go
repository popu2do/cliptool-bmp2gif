package session

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestFrameStoreAddPathsDeduplicatesByPath(t *testing.T) {
	store := NewFrameStore()
	imagePath := writeTestPNG(t, "one.png", color.White)

	result := store.AddPaths([]string{imagePath, imagePath})
	if result.Added != 1 {
		t.Fatalf("Added = %d, want 1", result.Added)
	}
	if result.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Skipped)
	}
	if len(result.Frames) != 1 {
		t.Fatalf("frame count = %d, want 1", len(result.Frames))
	}
}

func TestFrameStoreRemoveReorderAndClear(t *testing.T) {
	store := NewFrameStore()
	firstPath := writeTestPNG(t, "one.png", color.White)
	secondPath := writeTestPNG(t, "two.png", color.Black)
	thirdPath := writeTestPNG(t, "three.png", color.RGBA{R: 255, A: 255})

	result := store.AddPaths([]string{firstPath, secondPath, thirdPath})
	if len(result.Frames) != 3 {
		t.Fatalf("frame count = %d, want 3", len(result.Frames))
	}

	reordered := store.Reorder([]string{result.Frames[2].ID, result.Frames[0].ID, result.Frames[1].ID})
	if reordered[0].Path != thirdPath || reordered[1].Path != firstPath || reordered[2].Path != secondPath {
		t.Fatalf("unexpected reorder result: %+v", reordered)
	}

	remaining := store.Remove(reordered[1].ID)
	if len(remaining) != 2 {
		t.Fatalf("remaining count = %d, want 2", len(remaining))
	}
	if remaining[0].Path != thirdPath || remaining[1].Path != secondPath {
		t.Fatalf("unexpected remove result: %+v", remaining)
	}

	store.Clear()
	if frames := store.Frames(); len(frames) != 0 {
		t.Fatalf("frames after clear = %d, want 0", len(frames))
	}
}

func TestFrameStoreReorderKeepsMissingIDsAtEnd(t *testing.T) {
	store := NewFrameStore()
	firstPath := writeTestPNG(t, "one.png", color.White)
	secondPath := writeTestPNG(t, "two.png", color.Black)
	thirdPath := writeTestPNG(t, "three.png", color.RGBA{R: 255, A: 255})

	result := store.AddPaths([]string{firstPath, secondPath, thirdPath})
	reordered := store.Reorder([]string{result.Frames[2].ID, "missing", result.Frames[0].ID})

	if reordered[0].Path != thirdPath || reordered[1].Path != firstPath || reordered[2].Path != secondPath {
		t.Fatalf("unexpected reorder result: %+v", reordered)
	}
}

func TestFrameStoreCanAddPathAgainAfterRemove(t *testing.T) {
	store := NewFrameStore()
	imagePath := writeTestPNG(t, "one.png", color.White)

	result := store.AddPaths([]string{imagePath})
	if result.Added != 1 {
		t.Fatalf("Added = %d, want 1", result.Added)
	}

	store.Remove(result.Frames[0].ID)
	result = store.AddPaths([]string{imagePath})
	if result.Added != 1 {
		t.Fatalf("Added after remove = %d, want 1", result.Added)
	}
}

func writeTestPNG(t *testing.T, name string, c color.Color) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, c)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return path
}
