package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareDirectories(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	for _, root := range []string{left, right} {
		if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("---\nname: example\ndescription: test\n---\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(left, "changed.txt"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(right, "changed.txt"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(right, "new.bin"), []byte{0, 1}, 0600); err != nil {
		t.Fatal(err)
	}
	changes, leftDigest, rightDigest, err := CompareDirectories(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[0].Kind != "modified" || changes[0].LeftText != "old" || changes[0].RightText != "new" || changes[1].Kind != "added" || !changes[1].Binary || leftDigest == rightDigest {
		t.Fatalf("unexpected changes: %+v", changes)
	}
}

func TestCheckoutGitHubCleansUpOnFetchFailure(t *testing.T) {
	tmp, bin := t.TempDir(), t.TempDir()
	git := filepath.Join(bin, "git")
	if err := os.WriteFile(git, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmp)
	t.Setenv("PATH", bin)
	_, _, _, err := CheckoutGitHub(context.Background(), GitHubSource{RepositoryURL: "https://github.com/example/repo", Ref: "main"})
	if err == nil {
		t.Fatal("expected git failure")
	}
	entries, err := os.ReadDir(tmp)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary directory not cleaned: %+v, %v", entries, err)
	}
}
