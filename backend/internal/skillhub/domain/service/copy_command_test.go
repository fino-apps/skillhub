package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyCommand(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "snapshot 'one'")
	destination := filepath.Join(base, "destination")
	if err := os.MkdirAll(filepath.Join(source, "scripts"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "scripts", "run"), []byte("original"), 0700); err != nil {
		t.Fatal(err)
	}
	command, err := CopyCommand(source, "", "", "", "playwright")
	if err != nil {
		t.Fatal(err)
	}
	run := func() error { cmd := exec.Command("sh", "-c", command); cmd.Dir = destination; return cmd.Run() }
	if err := run(); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(destination, "playwright", "scripts", "run")
	if content, err := os.ReadFile(copied); err != nil || string(content) != "original" {
		t.Fatalf("copy: %q, %v", content, err)
	}
	if err := os.WriteFile(copied, []byte("existing"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("expected existing target to be rejected")
	}
	if content, _ := os.ReadFile(copied); string(content) != "existing" {
		t.Fatal("existing content was overwritten")
	}
}

func TestPublicCopyCommand(t *testing.T) {
	sha := strings.Repeat("a", 40)
	command, err := CopyCommand("", "https://github.com/owner/repo", "skills/playwright", sha, "playwright")
	if err != nil || !strings.Contains(command, "'owner/repo/skills/playwright#"+sha+"'") {
		t.Fatalf("%s: %v", command, err)
	}
	if _, err := CopyCommand("", "https://github.com/owner/repo", "../outside", sha, "playwright"); err == nil {
		t.Fatal("expected path rejection")
	}
}
