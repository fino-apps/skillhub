package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithinResolvesRootAlias(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "upload")
	want := filepath.Join(root, "sample")
	if err := os.MkdirAll(want, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Within(alias, "sample")
	if err != nil || got != want {
		t.Fatalf("Within(%q, sample) = %q, %v; want %q", alias, got, err, want)
	}
	if _, err := Within(alias, "../"); err == nil {
		t.Fatal("Within accepted a path outside the root")
	}
}
