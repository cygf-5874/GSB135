package tarcanon

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalBasic(t *testing.T) {
	data, err := Canonical([]Entry{
		{Path: "a.txt", Mode: 0o644, Body: []byte("hello")},
	})
	if err != nil {
		t.Fatalf("Canonical 返回错误：%v", err)
	}
	hdr, err := tar.NewReader(bytes.NewReader(data)).Next()
	if err != nil {
		t.Fatalf("读取失败：%v", err)
	}
	if hdr.Name != "a.txt" {
		t.Fatalf("Name = %q，期望 a.txt", hdr.Name)
	}
}

func TestCanonicalSorted(t *testing.T) {
	data, _ := Canonical([]Entry{
		{Path: "b.txt", Mode: 0o644, Body: []byte("b")},
		{Path: "a.txt", Mode: 0o644, Body: []byte("a")},
	})
	first, _ := tar.NewReader(bytes.NewReader(data)).Next()
	if first == nil || first.Name != "a.txt" {
		t.Fatalf("首个条目不是 a.txt")
	}
}

func TestCanonicalEmpty(t *testing.T) {
	data, err := Canonical(nil)
	if err != nil {
		t.Fatalf("空输入返回错误：%v", err)
	}
	if len(data) == 0 {
		t.Fatalf("空归档不应为零长度")
	}
}

func TestArchiveTempDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("准备样本失败：%v", err)
	}
	if _, err := Archive(dir); err != nil {
		t.Fatalf("Archive 返回错误：%v", err)
	}
}
