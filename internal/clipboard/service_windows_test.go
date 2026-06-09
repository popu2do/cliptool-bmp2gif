package clipboard

import "testing"

func TestCalcHashIsStableForSameFiles(t *testing.T) {
	files := []string{`C:\temp\a.bmp`, `C:\temp\b.bmp`}

	first := calcHash(files)
	second := calcHash([]string{`C:/TEMP/a.bmp`, `C:/TEMP/b.bmp`})

	if first != second {
		t.Fatal("calcHash() should stay stable for the same Windows file list")
	}
}
