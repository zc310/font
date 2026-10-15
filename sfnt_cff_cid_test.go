package font

import (
	"os"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseCIDFontDICTLegacyOperators(t *testing.T) {
	// Integer operands encode PaintType=2, CharstringType=2, and StrokeWidth=10.
	dict, err := parseFontDICT([]byte{141, 12, 5, 141, 12, 6, 149, 12, 8, 139, 139, 18}, false)
	if err != nil {
		t.Fatal(err)
	}
	if dict.PrivateLength != 0 || dict.PrivateOffset != 0 {
		t.Fatalf("private dict = (%d, %d), want (0, 0)", dict.PrivateLength, dict.PrivateOffset)
	}
	if dict.PaintType != 2 || dict.CharstringType != 2 || dict.StrokeWidth != 10 {
		t.Fatalf("Font DICT operators = paint %d, charstring %d, stroke %g", dict.PaintType, dict.CharstringType, dict.StrokeWidth)
	}

	if _, err := parsePrivateDICT([]byte{139, 12, 15, 139, 12, 16}, false); err != nil {
		t.Fatalf("legacy Type 1 private operators: %v", err)
	}
}

// TestCFFCIDMultiFDSubset keeps subsetting working for CID CFF fonts with multiple
// Font DICTs. These fonts previously failed with "only single-font CFFs are supported",
// preventing PDF font embedding and turning text into non-copyable vector paths.
func TestCFFCIDMultiFDSubset(t *testing.T) {
	b, err := os.ReadFile("resources/cid-cff-subset.otf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)
	if sfnt.CFF == nil || !sfnt.CFF.top.IsCID {
		t.Fatal("expected a CID CFF font")
	}
	if len(sfnt.CFF.fonts.private) < 2 {
		t.Fatalf("expected multiple Font DICTs, got %d", len(sfnt.CFF.fonts.private))
	}

	// 选取若干字形（.notdef 及用私有区映射到的 CID 字形）。
	glyphs := []uint16{0}
	for _, cid := range []uint16{19, 97, 1063, 1720, 2169, 4605} {
		if g := sfnt.GlyphIndex(rune(0xF0000 + int(cid))); g != 0 {
			glyphs = append(glyphs, g)
		}
	}

	subset, err := sfnt.Subset(glyphs, SubsetOptions{Tables: KeepAllTables})
	test.Error(t, err)
	test.T(t, subset.NumGlyphs(), uint16(len(glyphs)))

	out := subset.Write()
	reparsed, err := ParseSFNT(out, 0)
	test.Error(t, err)
	if reparsed.CFF == nil || !reparsed.CFF.top.IsCID {
		t.Fatal("subset output is not a CID CFF")
	}
	if len(reparsed.CFF.fonts.private) != len(sfnt.CFF.fonts.private) {
		t.Fatalf("subset lost Font DICTs: got %d, want %d", len(reparsed.CFF.fonts.private), len(sfnt.CFF.fonts.private))
	}
	test.T(t, reparsed.NumGlyphs(), uint16(len(glyphs)))
}
