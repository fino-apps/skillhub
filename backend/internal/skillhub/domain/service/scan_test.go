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

func TestScanProjectDirectSkill(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("---\nname: example\ndescription: Direct skill\n---\nContent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ScanProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "example" || got[0].Path != "." {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestScanProjectDirectSkillRejectsOutsideFile(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(outside, []byte("---\nname: example\ndescription: outside\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanProject(root); err == nil {
		t.Fatal("expected outside SKILL.md to be rejected")
	}
}

func TestScanProjectSkillCollectionAndCRLF(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	skill := filepath.Join(root, "example")
	if err := os.MkdirAll(skill, 0755); err != nil {
		t.Fatal(err)
	}
	content := "\ufeff---\r\nname: example\r\ndescription: Example skill\r\n---\r\nContent\r\n"
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ScanProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "example" || got[0].Path != "example" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestScanProjectDoesNotAcceptInvalidFrontmatterEnd(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("---\nname: example\ndescription: test\n---invalid\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ScanProject(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, error %v", got, err)
	}
}
