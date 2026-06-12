package text

import "testing"

func TestEncodeForPDFWindows1252(t *testing.T) {
	got := EncodeForPDF("• € ñ")
	want := string([]byte{0x95, 0x20, 0x80, 0x20, 0xF1})
	if got != want {
		t.Fatalf("EncodeForPDF() = %q, want %q", got, want)
	}
}

func TestEncodeForPDFReplacesUnsupportedRunes(t *testing.T) {
	got := EncodeForPDF("snowman ☃")
	want := "snowman ?"
	if got != want {
		t.Fatalf("EncodeForPDF() = %q, want %q", got, want)
	}
}
