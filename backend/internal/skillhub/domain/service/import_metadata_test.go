package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fino-io/finokit/db"
	domainrepo "github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo"
	sqlite "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/model/sqlite"
	"github.com/stretchr/testify/require"
)

func testSQLiteLibrary(t *testing.T) *Library {
	t.Helper()
	ctx := context.Background()
	provider, err := db.NewWithConfig(&db.Config{Driver: db.SqliteDriver, DSN: filepath.Join(t.TempDir(), "library.db")})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, provider.Close()) })
	require.NoError(t, provider.NewSession(ctx).AutoMigrate(&sqlite.Skill{}, &sqlite.SkillTag{}, &sqlite.SkillVersion{}, &sqlite.SkillSourceAlias{}))
	store := repo.NewLibrary(repo.LibraryDeps{DB: provider})
	return NewLibrary(LibraryDeps{Store: store, SnapshotDir: t.TempDir(), UploadDir: t.TempDir()})
}

func TestImportMetadataAppliesOnlyToNewSkill(t *testing.T) {
	ctx := context.Background()
	library := testSQLiteLibrary(t)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample\n---\n"), 0600))
	source := Source{
		Kind: "github", Repository: "https://github.com/openai/skills",
		InitialMetadata: Metadata{Category: "github.com/openai", Tags: []string{"docs"}},
	}
	version, created, err := library.saveDirectory(ctx, dir, "", source, VersionInput{})
	require.NoError(t, err)
	require.True(t, created)

	source.InitialMetadata = Metadata{Category: "Other", Tags: []string{"other"}}
	_, created, err = library.saveDirectory(ctx, dir, "", source, VersionInput{})
	require.NoError(t, err)
	require.False(t, created)

	details, err := library.Detail(ctx, version.SkillId)
	require.NoError(t, err)
	require.Equal(t, "github.com/openai", details.Skill.Category)
	require.Equal(t, source.Repository, details.Skill.UpstreamUrl)
	require.Equal(t, []string{"docs"}, details.Tags)
}

func TestArchiveImportSavesMetadataOnCreation(t *testing.T) {
	ctx := context.Background()
	library := testSQLiteLibrary(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: sample\n---\n"), 0600))
	archive, _, err := Archive(dir)
	require.NoError(t, err)
	upload, err := library.StageArchive(archive)
	require.NoError(t, err)
	require.Len(t, upload.Items, 1)
	paths := []string{upload.Items[0].Path}

	metadata := Metadata{
		Category: " Docs ", Purpose: " Summarize ", Notes: "Notes",
		UpstreamURL: "https://example.com/skill", Tags: []string{"docs", "docs"},
	}
	result, err := library.ImportArchive(ctx, upload.ID, "", paths, false, metadata)
	require.NoError(t, err)
	require.Empty(t, result.Failures)
	require.Equal(t, 1, result.Added)

	retry, err := library.ImportArchive(ctx, upload.ID, "", paths, false, Metadata{Category: "Other"})
	require.NoError(t, err)
	require.Equal(t, 1, retry.Existing)

	skills, _, err := library.Skills(ctx, domainrepo.SkillFilter{}, 10, "")
	require.NoError(t, err)
	require.Len(t, skills, 1)
	require.Equal(t, "Docs", skills[0].Skill.Category)
	require.Equal(t, "Summarize", skills[0].Skill.Purpose)
	require.Equal(t, metadata.Notes, skills[0].Skill.Notes)
	require.Equal(t, metadata.UpstreamURL, skills[0].Skill.UpstreamUrl)
	require.Equal(t, []string{"docs"}, skills[0].Tags)
}
