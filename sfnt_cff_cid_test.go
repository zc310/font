package font

import (
	"os"
	"testing"

	"github.com/tdewolff/test"
)

// TestCFFCIDMultiFDSubset 保护多 Font DICT（CID）CFF 的子集化：这类字体此前
// 会因 "only single-font CFFs are supported" 失败，导致 PDF 无法嵌入字体、
// 文字退化为矢量路径而不可复制。
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
