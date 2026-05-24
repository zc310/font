package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/tdewolff/font"
)

type Subset struct {
	Quiet         bool     `short:"q" desc:"Suppress output except for errors."`
	Force         bool     `short:"f" desc:"Force overwriting existing files."`
	Glyphs        []string `short:"g" name:"glyph" desc:"List of glyph IDs to keep, eg. 1-100."`
	Chars         []string `short:"c" name:"char" desc:"List of literal characters to keep, eg. a-z. The same escape sequences are supported as for Go strings, where , and - also need escaping."`
	Names         []string `short:"n" name:"name" desc:"List of glyph names to keep, eg. space."`
	Unicodes      []string `short:"u" name:"unicode" desc:"List of unicode IDs to keep, eg. f0fc-f0ff."`
	UnicodeRanges []string `short:"r" name:"range" desc:"List of unicode categories or scripts to keep, eg. L (for Letters) or Latin (latin script). See https://pkg.go.dev/unicode for all supported values."`
	Index         int      `short:"i" desc:"Index into font collection, used with TTC or OTC."`
	Type          string   `short:"t" desc:"Explicitly set output mimetype, eg. font/woff2."`
	Encoding      string   `short:"e" desc:"Output encoding, either empty of base64."`
	GlyphName     string   `desc:"New glyph name. Available variables: %i glyph ID, %n glyph name, %u glyph unicode in hexadecimal."`
	RearrangeCmap bool     `desc:"Rearrange glyph unicode mapping, assigning a sequential codepoint for each glyph in order starting at 33 (exclamation)."`
	Outputs       []string `short:"o" desc:"Output font files, only TTF/OTF/WOFF2/TTC/OTC are supported."`
	Inputs        []string `index:"*" desc:"Input font files, multiple fallback fonts are supported."`
}

func (cmd *Subset) Run() error {
	if cmd.Quiet {
		Warning = log.New(ioutil.Discard, "", 0)
	}

	if len(cmd.Inputs) == 0 {
		return fmt.Errorf("missing input font file")
	} else if len(cmd.Outputs) == 0 {
		if 1 < len(cmd.Inputs) {
			return fmt.Errorf("missing output font file")
		}
		cmd.Outputs = []string{cmd.Inputs[0]}
	} else if cmd.Encoding != "" && cmd.Encoding != "base64" {
		return fmt.Errorf("unsupported encoding: %v", cmd.Encoding)
	}

	// read from file and parse font
	sfnt, rMimetype, rLen, err := readFont(cmd.Inputs[0], cmd.Index)
	if err != nil {
		if cmd.Inputs[0] == "-" {
			return err
		}
		return fmt.Errorf("%v: %v", cmd.Inputs[0], err)
	}
	numGlyphs := sfnt.NumGlyphs()

	sfnts := []*font.SFNT{sfnt}
	glyphMaps := []map[uint16]bool{
		{},
	}
	for _, input := range cmd.Inputs[1:] {
		sfnt, _, _, err := readFont(input, 0)
		if err != nil {
			if input == "-" {
				return err
			}
			return fmt.Errorf("%v: %v", input, err)
		}
		sfnts = append(sfnts, sfnt)
		glyphMaps = append(glyphMaps, map[uint16]bool{})
	}

	// append glyphs
	for _, glyph := range cmd.Glyphs {
		if dash := strings.IndexByte(glyph, '-'); dash != -1 {
			first, err := strconv.ParseInt(glyph[:dash], 10, 16)
			if err != nil {
				return fmt.Errorf("invalid glyph ID: %v", err)
			}
			last, err := strconv.ParseInt(glyph[dash+1:], 10, 16)
			if err != nil {
				return fmt.Errorf("invalid glyph ID: %v", err)
			}
			if last < first || first < 0 || 65535 < last {
				return fmt.Errorf("invalid glyph ID range: %d-%d\n", first, last)
			}
			for first != last+1 {
				glyphMaps[0][uint16(first)] = true
				first++
			}
		} else {
			glyphID, err := strconv.ParseInt(glyph, 10, 16)
			if err != nil {
				return fmt.Errorf("invalid glyph ID: %v", err)
			}
			if glyphID < 0 || 65535 < glyphID {
				return fmt.Errorf("invalid glyph ID: %v", glyphID)
			}
			glyphMaps[0][uint16(glyphID)] = true
		}
	}

	// append characters
	for _, s := range cmd.Chars {
		prev := rune(-1)
		rangeChars := false
		runes := []rune(s)
		for i := 0; i < len(runes); i++ {
			r := runes[i]
			escaped := false
			if r == '\\' && i+1 < len(runes) {
				switch runes[i+1] {
				case 'a':
					r = '\u0007'
					i++
				case 'b':
					r = '\u0008'
					i++
				case 't':
					r = '\u0009'
					i++
				case 'n':
					r = '\u000A'
					i++
				case 'v':
					r = '\u000B'
					i++
				case 'f':
					r = '\u000C'
					i++
				case 'r':
					r = '\u000D'
					i++
				case '"':
					r = '\u0022'
					i++
				case '\'':
					r = '\u0027'
					i++
				case ',':
					r = '\u002C'
					i++
				case '-':
					escaped = true
					r = '\u002D'
					i++
				case '\\':
					r = '\u005C'
					i++
				case 'x':
					if i+3 < len(runes) {
						if h, ok := parseHexRunes(runes[i+2 : i+4]); ok {
							r = rune(h)
							i += 3
						} else {
							Warning.Println("invalid escape sequence:", string(runes[i:i+4]))
						}
					} else {
						Warning.Println("invalid escape sequence:", string(runes[i:i+4]))
					}
				case 'u':
					if i+5 < len(runes) {
						if h, ok := parseHexRunes(runes[i+2 : i+6]); ok {
							r = rune(h)
							i += 5
						} else {
							Warning.Println("invalid escape sequence:", string(runes[i:i+6]))
						}
					} else {
						Warning.Println("invalid escape sequence:", string(runes[i:i+6]))
					}
				case 'U':
					if i+9 < len(runes) {
						if h, ok := parseHexRunes(runes[i+2 : i+10]); ok {
							r = rune(h)
							i += 9
						} else {
							Warning.Println("invalid escape sequence:", string(runes[i:i+10]))
						}
					} else {
						Warning.Println("invalid escape sequence:", string(runes[i:i+10]))
					}
				default:
					Warning.Println("invalid escape sequence:", string(runes[i:i+2]))
				}
			}
			if prev != -1 && r == '-' && !escaped {
				rangeChars = true
			} else if rangeChars {
				for i := prev + 1; i <= r; i++ {
					k, glyphID := 0, uint16(0)
					for k < len(sfnts) {
						glyphID = sfnts[k].GlyphIndex(i)
						if glyphID != 0 {
							break
						}
						k++
					}
					if glyphID == 0 {
						Warning.Println("glyph not found:", printableRune(i))
					} else {
						glyphMaps[k][glyphID] = true
					}
				}
				rangeChars = false
				prev = -1
			} else {
				k, glyphID := 0, uint16(0)
				for k < len(sfnts) {
					glyphID = sfnts[k].GlyphIndex(r)
					if glyphID != 0 {
						break
					}
					k++
				}
				if glyphID == 0 {
					Warning.Println("glyph not found:", printableRune(r))
				} else {
					glyphMaps[k][glyphID] = true
				}
				prev = r
			}
		}
		if rangeChars {
			k, glyphID := 0, uint16(0)
			for k < len(sfnts) {
				glyphID = sfnts[k].GlyphIndex('-')
				if glyphID != 0 {
					break
				}
				k++
			}
			if glyphID == 0 {
				Warning.Println("glyph not found: -")
			} else {
				glyphMaps[k][glyphID] = true
			}
		}
	}

	// append glyph names
	for _, name := range cmd.Names {
		k, glyphID := 0, uint16(0)
		for k < len(sfnts) {
			glyphID = sfnts[k].FindGlyphName(name)
			if glyphID != 0 {
				break
			}
			k++
		}
		if glyphID == 0 {
			Warning.Println("glyph name not found:", name)
		} else {
			glyphMaps[k][glyphID] = true
		}
	}

	// append unicode
	for _, code := range cmd.Unicodes {
		if dash := strings.IndexByte(code, '-'); dash != -1 {
			first, err := strconv.ParseInt(code[:dash], 16, 32)
			if err != nil {
				return fmt.Errorf("invalid unicode codepoint: %v", err)
			}
			last, err := strconv.ParseInt(code[dash+1:], 16, 32)
			if err != nil {
				return fmt.Errorf("invalid unicode codepoint: %v", err)
			}
			if last < first || first < 0 {
				return fmt.Errorf("invalid unicode range: U+%4X-U+%4X\n", first, last)
			}
			for first != last+1 {
				k, glyphID := 0, uint16(0)
				for k < len(sfnts) {
					glyphID = sfnts[k].GlyphIndex(rune(first))
					if glyphID != 0 {
						break
					}
					k++
				}
				if glyphID == 0 {
					Warning.Printf("glyph not found for U+%4X\n", first)
				} else {
					glyphMaps[k][glyphID] = true
				}
				first++
			}
		} else {
			codepoint, err := strconv.ParseInt(code, 16, 32)
			if err != nil {
				return fmt.Errorf("invalid unicode codepoint: %v", err)
			} else if codepoint < 0 {
				return fmt.Errorf("invalid unicode codepoint: U+%4X\n", codepoint)
			}
			k, glyphID := 0, uint16(0)
			for k < len(sfnts) {
				glyphID = sfnts[k].GlyphIndex(rune(codepoint))
				if glyphID != 0 {
					break
				}
				k++
			}
			if glyphID == 0 {
				Warning.Printf("glyph not found for U+%4X\n", codepoint)
			} else {
				glyphMaps[k][glyphID] = true
			}
		}
	}

	// append unicode ranges
	for _, unicodeRange := range cmd.UnicodeRanges {
		var ok bool
		var table *unicode.RangeTable
		if table, ok = unicode.Categories[unicodeRange]; !ok {
			if table, ok = unicode.Scripts[unicodeRange]; !ok {
				return fmt.Errorf("invalid unicode range: %v", unicodeRange)
			}
		}
		for _, ran := range table.R16 {
			for r := ran.Lo; r <= ran.Hi; r += ran.Stride {
				k, glyphID := 0, uint16(0)
				for k < len(sfnts) {
					glyphID = sfnts[k].GlyphIndex(rune(r))
					if glyphID != 0 {
						break
					}
					k++
				}
				if glyphID != 0 {
					glyphMaps[k][glyphID] = true
				}

			}
		}
		for _, ran := range table.R32 {
			for r := ran.Lo; r <= ran.Hi; r += ran.Stride {
				k, glyphID := 0, uint16(0)
				for k < len(sfnts) {
					glyphID = sfnts[k].GlyphIndex(rune(r))
					if glyphID != 0 {
						break
					}
					k++
				}
				if glyphID != 0 {
					glyphMaps[k][glyphID] = true
				}
			}
		}
	}

	var subset *font.SFNT
	options := font.MergeOptions{
		RearrangeCmap: cmd.RearrangeCmap,
	}
	for k, glyphMap := range glyphMaps {
		if len(glyphMap) == 0 {
			continue
		}
		glyphMap[0] = true

		// convert to sorted list, prevents duplicates
		glyphIDs := make([]uint16, 0, len(glyphMap))
		for glyphID := range glyphMap {
			glyphIDs = append(glyphIDs, glyphID)
		}
		sort.Slice(glyphIDs, func(i, j int) bool { return glyphIDs[i] < glyphIDs[j] })

		if sfnts[k].IsCFF && cmd.GlyphName == "" {
			sfnts[k].CFF.SetGlyphNames(nil)
		}

		// subset font
		sfntSubset, err := sfnts[k].Subset(glyphIDs, font.SubsetOptions{Tables: font.KeepMinTables})
		if err != nil {
			if cmd.Inputs[k] == "-" {
				return err
			}
			return fmt.Errorf("%v: %v", cmd.Inputs[k], err)
		}
		if cmd.GlyphName != "" && cmd.GlyphName != "%n" {
			names := make([]string, len(glyphIDs))
			for glyphID := range glyphIDs {
				name, ok := fmtName(cmd.GlyphName, sfntSubset, uint16(glyphID))
				if !ok {
					Warning.Printf("%v: missing glyph name or unicode mapping for glyph: %s(%d)", cmd.Inputs[k], sfnts[k].GlyphName(uint16(glyphID)), glyphID)
				} else {
					names[glyphID] = name
				}
			}
			if err := sfntSubset.SetGlyphNames(names); err != nil {
				return fmt.Errorf("glyph names: %v", err)
			}
		}
		if subset == nil {
			subset = sfntSubset
		} else if err := subset.Merge(sfntSubset, options); err != nil {
			if cmd.Inputs[k] == "-" {
				return err
			}
			return fmt.Errorf("%v: %v", cmd.Inputs[k], err)
		}
	}
	if subset == nil {
		return fmt.Errorf("output is empty")
	}

	// create font program
	for _, output := range cmd.Outputs {
		mimetype := extMimetype[filepath.Ext(output)]
		if cmd.Type != "" {
			mimetype = cmd.Type
		} else if mimetype == "" {
			mimetype = rMimetype
		}
		wLen, err := writeFont(output, mimetype, cmd.Encoding, cmd.Force, subset)
		if err != nil {
			return err
		}

		ratio := 1.0
		if 0 < rLen {
			ratio = float64(wLen) / float64(rLen)
		}
		if !cmd.Quiet && output != "-" {
			numGlyphsSubset := subset.NumGlyphs()
			fmt.Printf("%v:  %v => %v glyphs,  %v => %v (%.1f%%)\n", filepath.Base(output), numGlyphs, numGlyphsSubset, formatBytes(uint64(rLen)), formatBytes(uint64(wLen)), ratio*100.0)
		}
	}
	return nil
}
