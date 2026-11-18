package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCleanUploads(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	old := uuid.NewString()
	recent := uuid.NewString()
	for _, name := range []string{old, recent, "unrelated"} {
		path := filepath.Join(dir, name)
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		modified := now
		if name != recent {
			modified = now.Add(-25 * time.Hour)
		}
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	library := NewLibrary(LibraryDeps{UploadDir: dir})
	if err := library.cleanUploads(now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, old)); !os.IsNotExist(err) {
		t.Fatalf("expired upload remains: %v", err)
	}
	for _, name := range []string{recent, "unrelated"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s should remain: %v", name, err)
		}
	}
	if err := library.cleanUploads(now); err != nil {
		t.Fatalf("repeated cleanup: %v", err)
	}
}
