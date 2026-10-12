package font

import (
	"os"
	"testing"

	"github.com/tdewolff/test"
)

// TestCFFWriteGeneratesCharsetWhenNamesCleared 保护：调用方（如 canvas 为压缩
// 体积）调用 SetGlyphNames(nil) 清空字形名后，CFF 字形数超过 229（charset
// format 0 的上限）时仍能写出 CFF。此前这种情况会直接报
// "charset must be set explicitly for more than 229 glyphs"，导致内嵌 CFF 无法
// 子集化/写出、文字退化为矢量轮廓而不可复制。
func TestCFFWriteGeneratesCharsetWhenNamesCleared(t *testing.T) {
	b, err := os.ReadFile("resources/EBGaramond12-Regular.otf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)
	if sfnt.CFF == nil || sfnt.CFF.top.IsCID {
		t.Fatal("expected a non-CID CFF font")
	}
	if sfnt.NumGlyphs() <= 229 {
		t.Fatalf("fixture needs >229 glyphs, got %d", sfnt.NumGlyphs())
	}

	sfnt.CFF.SetGlyphNames(nil)

	out, err := sfnt.CFF.Write()
	test.Error(t, err)
	if len(out) == 0 {
		t.Fatal("empty CFF output")
	}
}
