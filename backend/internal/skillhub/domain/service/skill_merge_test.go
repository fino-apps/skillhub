package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/stretchr/testify/require"
)

func TestMergeRepairsDefaultAndKeepsSourceAlias(t *testing.T) {
	ctx := t.Context()
	library := testSQLiteLibrary(t)
	const manifest = "---\nname: sample\ndescription: Sample\n---\n"
	create := func(notes string) (model.SkillVersionSnapshot, string) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(manifest), 0600))
		version, created, err := library.saveDirectory(ctx, dir, "", Source{Kind: "folder"}, VersionInput{Notes: notes})
		require.NoError(t, err)
		require.True(t, created)
		return version, dir
	}
	main, mainDir := create("main")
	incoming, incomingDir := create("history")
	// A usable sibling with identical content must not mask damage in the default version.
	sibling, created, err := library.saveDirectory(ctx, mainDir, main.SkillId, Source{
		Kind: "github", Commit: strings.Repeat("a", 40),
	}, VersionInput{})
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, main.ContentDigest, sibling.ContentDigest)
	require.NoError(t, os.WriteFile(library.snapshotPath(string(main.ID)), []byte("bad zip"), 0600))

	preview, err := library.MergeSkills(ctx, incoming.SkillId, main.SkillId, "")
	require.NoError(t, err)
	require.EqualValues(t, 1, preview.Restored)
	require.NotEmpty(t, preview.Revision)
	require.Len(t, preview.Changes, 1)
	require.Equal(t, "history", preview.Changes[0].Version.Notes)
	require.Equal(t, []string{string(main.ID)}, preview.Changes[0].Restored)

	_, err = library.ReplaceMetadata(ctx, incoming.SkillId, Metadata{Notes: "changed after preview"})
	require.NoError(t, err)
	_, err = library.MergeSkills(ctx, incoming.SkillId, main.SkillId, preview.Revision)
	require.ErrorIs(t, err, ErrConflict)
	require.False(t, library.usableSnapshot(main))
	require.True(t, library.usableSnapshot(incoming))
	_, err = library.Skill(ctx, incoming.SkillId)
	require.NoError(t, err)

	preview, err = library.MergeSkills(ctx, incoming.SkillId, main.SkillId, "")
	require.NoError(t, err)
	_, err = library.MergeSkills(ctx, incoming.SkillId, main.SkillId, preview.Revision)
	require.NoError(t, err)
	require.True(t, library.usableSnapshot(main))
	require.NoFileExists(t, library.snapshotPath(string(incoming.ID)))
	primary, err := library.Skill(ctx, main.SkillId)
	require.NoError(t, err)
	require.Equal(t, string(main.ID), primary.DefaultVersionId)
	_, err = library.Skill(ctx, incoming.SkillId)
	require.ErrorIs(t, err, repo.ErrNotFound)

	backup, err := library.ExportBackup(ctx)
	require.NoError(t, err)
	restored := testSQLiteLibrary(t)
	_, err = restored.ImportBackup(ctx, backup)
	require.NoError(t, err)
	previewImport, err := restored.PreviewBackupImport(ctx, backup)
	require.NoError(t, err)
	require.Len(t, previewImport.Items, 1)
	require.Equal(t, main.SkillId, previewImport.Items[0].ExactTargetID)
	for _, catalog := range []*Library{library, restored} {
		_, created, err := catalog.saveDirectory(ctx, incomingDir, "", Source{Kind: "folder"}, VersionInput{})
		require.NoError(t, err)
		require.False(t, created)
	}
}

func TestSnapshotChangesPreserveReplacedFileOnRollback(t *testing.T) {
	library := NewLibrary(LibraryDeps{SnapshotDir: t.TempDir()})
	id := model.SkillVersionID("sample")
	path := library.snapshotPath(string(id))
	require.NoError(t, os.WriteFile(path, []byte("old"), 0600))
	changes := snapshotChanges{library: library}
	require.NoError(t, changes.write(id, []byte("new")))
	require.ErrorIs(t, changes.finish(ErrConflict), ErrConflict)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "old", string(data))
}
