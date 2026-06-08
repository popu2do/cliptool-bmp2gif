package clipboard

import "testing"

func TestCalcHashIncludesClipboardSequenceNumber(t *testing.T) {
	files := []string{`C:\temp\a.bmp`, `C:\temp\b.bmp`}

	first := calcHash(files, 1)
	second := calcHash(files, 2)

	if first == second {
		t.Fatal("calcHash() should change when clipboard sequence number changes")
	}
}
