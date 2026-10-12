package font

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetSFNTMetadataUsesOS2Weight 保护“Arial Black 被误判为 Arial 常规”的回归。
// ariblk.ttf 的 name 表把 subfamily 写成 "Regular"，只有读取 OS/2 的
// usWeightClass(900) 才能与真正的 Arial 常规（400）区分开；否则系统字体匹配
// 会用 Arial Black 覆盖 Arial 常规，导致文字被替换成更宽的字形而重叠。
func TestGetSFNTMetadataUsesOS2Weight(t *testing.T) {
	black := findSystemFontFile("ariblk.ttf")
	regular := findSystemFontFile("arial.ttf")
	if black == "" || regular == "" {
		t.Skip("Arial/Arial Black 未安装")
	}

	blackMeta := readSFNTMetadata(t, black)
	if got := blackMeta.Style.Weight(); got != Black {
		t.Fatalf("Arial Black weight = %v, want Black", got)
	}
	regularMeta := readSFNTMetadata(t, regular)
	if got := regularMeta.Style.Weight(); got != Regular {
		t.Fatalf("Arial weight = %v, want Regular", got)
	}
	if blackMeta.Style == regularMeta.Style {
		t.Fatalf("Arial Black and Arial map to the same style %v", blackMeta.Style)
	}
}

func readSFNTMetadata(t *testing.T, path string) FontMetadata {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	metadata, err := getSFNTMetadata(file)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return metadata
}

func findSystemFontFile(name string) string {
	for _, dir := range DefaultFontDirs() {
		var found string
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || found != "" {
				return nil
			}
			if entry.Name() == name {
				found = path
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}
