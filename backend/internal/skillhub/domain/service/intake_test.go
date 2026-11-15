package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

type testIntakeRepo struct {
	skill   *model.Skill
	version *model.SkillVersion
	fail    bool
}

func (r *testIntakeRepo) CreateInitial(_ context.Context, skill *model.Skill, version *model.SkillVersion, publish func() error) error {
	if err := publish(); err != nil {
		return err
	}
	if r.fail {
		return errors.New("database failure")
	}
	r.skill, r.version = skill, version
	return nil
}
func TestImportLocalSnapshot(t *testing.T) {
	source := t.TempDir()
	data := t.TempDir()
	for path, body := range map[string]string{"SKILL.md": "---\nname: sample\ndescription: a skill\n---\nBody\n", "scripts/run.sh": "echo hi", ".env": "SECRET=secret"} {
		filename := filepath.Join(source, path)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	files, digest, err := PreviewLocalSkill(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 || !files[0].Sensitive {
		t.Fatalf("preview: %+v", files)
	}
	repository := &testIntakeRepo{}
	intake := NewIntake(data, repository)
	skill, version, err := intake.ImportLocal(context.Background(), source, digest, []string{"SKILL.md", "scripts/run.sh"}, "1.0.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, skill.ContentPath(), "scripts/run.sh")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, version.SnapshotPath(), "scripts/run.sh")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, skill.ContentPath(), ".env")); !os.IsNotExist(err) {
		t.Fatalf("secret copied: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := intake.ImportLocal(context.Background(), source, digest, []string{"SKILL.md"}, "1.0.0", ""); err == nil {
		t.Fatal("expected stale preview to fail")
	}
}

func TestImportLocalRollsBackPublishedFiles(t *testing.T) {
	source, data := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: rollback\ndescription: test\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, digest, err := PreviewLocalSkill(source)
	if err != nil {
		t.Fatal(err)
	}
	intake := NewIntake(data, &testIntakeRepo{fail: true})
	if _, _, err := intake.ImportLocal(context.Background(), source, digest, []string{"SKILL.md"}, "1.0.0", ""); err == nil {
		t.Fatal("expected failure")
	}
	entries, err := os.ReadDir(filepath.Join(data, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("published files were not removed: %+v", entries)
	}
}
