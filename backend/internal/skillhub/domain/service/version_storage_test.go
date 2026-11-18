package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

type existingVersionStore struct {
	repo.Library
	tx existingVersionTx
}

type existingVersionTx struct {
	repo.LibraryTx
	skill   model.SkillSnapshot
	version model.SkillVersionSnapshot
}

func (s existingVersionStore) Transaction(_ context.Context, run func(repo.LibraryTx) error) error {
	return run(s.tx)
}

func (s existingVersionTx) Skill(string) (model.SkillSnapshot, error) {
	return s.skill, nil
}

func (s existingVersionTx) SkillBySource(string) (model.SkillSnapshot, error) {
	return s.skill, nil
}

func (s existingVersionTx) MatchingVersion(string, string, string) (model.SkillVersionSnapshot, error) {
	return s.version, nil
}

func TestSaveDirectoryRestoresExistingSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, digest, err := Archive(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	store := existingVersionStore{tx: existingVersionTx{
		skill:   model.SkillSnapshot{ID: model.SkillID(uuid.NewString()), DefaultVersionId: id},
		version: model.SkillVersionSnapshot{ID: model.SkillVersionID(id), ContentDigest: digest},
	}}
	for _, target := range []string{"", string(store.tx.skill.ID)} {
		library := NewLibrary(LibraryDeps{Store: store, SnapshotDir: t.TempDir()})
		version, created, err := library.saveDirectory(context.Background(), dir, target, Source{Kind: "folder"}, VersionInput{})
		if err != nil || created || version.ID != store.tx.version.ID {
			t.Fatalf("saveDirectory target=%q: version=%v created=%v err=%v", target, version.ID, created, err)
		}
		data, err := os.ReadFile(library.snapshotPath(id))
		if err != nil {
			t.Fatal(err)
		}
		extracted := t.TempDir()
		if err := Unpack(data, extracted); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadManifest(extracted); err != nil {
			t.Fatal(err)
		}
	}
}
