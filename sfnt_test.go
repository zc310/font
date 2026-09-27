package font

import (
	"encoding/binary"
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

// TestSFNTToleratesHmtxTrailingBytes checks that a font whose hmtx table has
// extra trailing bytes beyond what hhea/maxp require (e.g. numberOfHMetrics is
// short by one entry, as seen in some OFD subset fonts) is not rejected as a
// whole because of the harmless alignment padding.
func TestSFNTToleratesHmtxTrailingBytes(t *testing.T) {
	b, err := ioutil.ReadFile("resources/DejaVuSerif.ttf")
	test.Error(t, err)

	numTables := int(binary.BigEndian.Uint16(b[4:6]))
	hmtx := -1
	var insertAt int
	for i := 0; i < numTables; i++ {
		off := 12 + i*16
		if string(b[off:off+4]) != "hmtx" {
			continue
		}
		tableOffset := int(binary.BigEndian.Uint32(b[off+8 : off+12]))
		tableLength := int(binary.BigEndian.Uint32(b[off+12 : off+16]))
		insertAt = tableOffset + tableLength
		binary.BigEndian.PutUint32(b[off+12:off+16], uint32(tableLength+2))
		hmtx = off
		break
	}
	if hmtx < 0 {
		t.Fatal("hmtx table not found")
	}

	inserted := append(append(append([]byte{}, b[:insertAt]...), 0, 0), b[insertAt:]...)
	for i := 0; i < numTables; i++ {
		off := 12 + i*16
		tableOffset := int(binary.BigEndian.Uint32(inserted[off+8 : off+12]))
		if tableOffset >= insertAt {
			binary.BigEndian.PutUint32(inserted[off+8:off+12], uint32(tableOffset+2))
		}
	}

	sfnt, err := ParseSFNT(inserted, 0)
	test.Error(t, err)
	test.T(t, sfnt.NumGlyphs(), uint16(3528))
}

func TestSFNTDejaVuSerifTTF(t *testing.T) {
	b, err := ioutil.ReadFile("resources/DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)

	test.T(t, sfnt.Head.UnitsPerEm, uint16(2048))
	test.T(t, sfnt.Hhea.Ascender, int16(1901))
	test.T(t, sfnt.Hhea.Descender, int16(-483))
	test.T(t, sfnt.OS2.SCapHeight, int16(1493)) // height of H glyph
	test.T(t, sfnt.Head.XMin, int16(-1576))
	test.T(t, sfnt.Head.YMin, int16(-710))
	test.T(t, sfnt.Head.XMax, int16(4312))
	test.T(t, sfnt.Head.YMax, int16(2272))

	id := sfnt.GlyphIndex(' ')
	contour, err := sfnt.Glyf.Contour(id)
	test.Error(t, err)
	test.T(t, contour.GlyphID, id)
	test.T(t, len(contour.XCoordinates), 0)
}

func TestSFNTWrite(t *testing.T) {
	b, err := ioutil.ReadFile("resources/DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)

	b2 := sfnt.Write()
	sfnt2, err := ParseSFNT(b2, 0)
	test.Error(t, err)

	test.T(t, sfnt2.GlyphIndex('A'), sfnt.GlyphIndex('A'))
	test.T(t, sfnt2.GlyphIndex('B'), sfnt.GlyphIndex('B'))
	test.T(t, sfnt2.GlyphIndex('C'), sfnt.GlyphIndex('C'))

	//ioutil.WriteFile("out.otf", subset, 0644)
}

func TestSFNTSubset(t *testing.T) {
	b, err := ioutil.ReadFile("resources/DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)

	sfntSubset, err := sfnt.Subset([]uint16{0, 3, 6, 36, 37, 38, 55, 131}, SubsetOptions{Tables: KeepAllTables}) // .notdef, space, #, A, B, C, T, Á
	test.Error(t, err)

	test.T(t, sfntSubset.NumGlyphs(), uint16(9)) // Á is a composite glyph containing two simple glyphs: 36 and 3452

	test.T(t, sfntSubset.GlyphIndex('A'), uint16(3))
	test.T(t, sfntSubset.GlyphIndex('B'), uint16(4))
	test.T(t, sfntSubset.GlyphIndex('C'), uint16(5))

	//ioutil.WriteFile("out.otf", subset, 0644)
}
