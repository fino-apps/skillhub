package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanProjectDeduplicatesLinkedSkillDirectories(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "skills", "example")
	if err := os.MkdirAll(skill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: example\ndescription: An example\n---\nContent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{".agents", ".codex"} {
		if err := os.MkdirAll(filepath.Join(root, folder), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, "skills"), filepath.Join(root, folder, "skills")); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ScanProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "example" || got[0].Path != "skills/example" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestScanProjectRejectsOutsideLink(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".agents", "skills")); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanProject(root); err == nil {
		t.Fatal("expected outside link to be rejected")
	}
}
