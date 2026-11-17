package service

import (
	"errors"
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
