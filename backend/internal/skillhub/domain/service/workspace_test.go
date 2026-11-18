package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDirectoryRejectsRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LocalDirectory(path); !errors.Is(err, ErrInvalid) {
		t.Fatalf("LocalDirectory(file) error = %v, want ErrInvalid", err)
	}
}

func TestScanMoreThanImportBatch(t *testing.T) {
	dir := t.TempDir()
	for i := range 101 {
		path := filepath.Join(dir, fmt.Sprint(i))
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample\n---\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	items, err := Scan(dir)
	if err != nil || len(items) != 101 {
		t.Fatalf("Scan() = %d items, %v, want 101 items", len(items), err)
	}
}
