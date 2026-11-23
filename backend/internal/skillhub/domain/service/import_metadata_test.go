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
	for _, tc := range []struct {
		kind            string
		directoryUpload bool
	}{
		{kind: "zip"},
		{kind: "directory_upload", directoryUpload: true},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			testArchiveImport(t, tc.kind, tc.directoryUpload)
		})
	}
}

func testArchiveImport(t *testing.T, kind string, directoryUpload bool) {
	t.Helper()
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
		UpstreamName: " Team library ", UpstreamURL: "https://example.com/skill", Tags: []string{"docs", "docs"},
	}
	result, err := library.ImportArchive(ctx, upload.ID, "", paths, ArchiveImportOptions{DirectoryUpload: directoryUpload, Metadata: metadata})
	require.NoError(t, err)
	require.Empty(t, result.Failures)
	require.Equal(t, 1, result.Added)

	retry, err := library.ImportArchive(ctx, upload.ID, "", paths, ArchiveImportOptions{DirectoryUpload: !directoryUpload, Metadata: Metadata{Category: "Other", UpstreamName: "Other source"}})
	require.NoError(t, err)
	require.Equal(t, 1, retry.Existing)

	skills, _, err := library.Skills(ctx, domainrepo.SkillFilter{}, 10, "")
	require.NoError(t, err)
	require.Len(t, skills, 1)
	require.Equal(t, kind, skills[0].Skill.ImportKind)
	require.Equal(t, "Docs", skills[0].Skill.Category)
	require.Equal(t, "Summarize", skills[0].Skill.Purpose)
	require.Equal(t, metadata.Notes, skills[0].Skill.Notes)
	require.Equal(t, "Team library", skills[0].Skill.UpstreamName)
	require.Equal(t, metadata.UpstreamURL, skills[0].Skill.UpstreamUrl)
	require.Equal(t, []string{"docs"}, skills[0].Tags)
}

func TestDirectorySourcesShareLibraryFilter(t *testing.T) {
	ctx := context.Background()
	library := testSQLiteLibrary(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: sample\n---\n"), 0600))

	ids := make(map[string]string)
	for _, kind := range []string{"folder", "directory_upload", "github"} {
		version, created, err := library.saveDirectory(ctx, dir, "", Source{Kind: kind}, VersionInput{})
		require.NoError(t, err)
		require.True(t, created)
		ids[kind] = version.SkillId
	}

	facets, err := library.Facets(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"folder", "github"}, facets.Sources)
	require.Equal(t, []string{"GitHub", "本机目录"}, facets.SourceNames)

	for _, filter := range []string{"folder", "directory_upload"} {
		skills, _, err := library.Skills(ctx, domainrepo.SkillFilter{Source: filter}, 10, "")
		require.NoError(t, err)
		require.Len(t, skills, 2)
		require.ElementsMatch(t, []string{"folder", "directory_upload"}, []string{skills[0].Skill.ImportKind, skills[1].Skill.ImportKind})
	}

	_, err = library.ReplaceMetadata(ctx, ids["directory_upload"], Metadata{UpstreamName: "github"})
	require.NoError(t, err)
	facets, err = library.Facets(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"GitHub", "github", "本机目录"}, facets.SourceNames)

	for name, kind := range map[string]string{"本机目录": "folder", "github": "directory_upload", "GitHub": "github"} {
		skills, _, err := library.Skills(ctx, domainrepo.SkillFilter{SourceName: name}, 10, "")
		require.NoError(t, err)
		require.Len(t, skills, 1)
		require.Equal(t, ids[kind], string(skills[0].Skill.ID))
	}

	_, err = library.ReplaceMetadata(ctx, ids["directory_upload"], Metadata{})
	require.NoError(t, err)
	facets, err = library.Facets(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"GitHub", "本机目录"}, facets.SourceNames)
	skills, _, err := library.Skills(ctx, domainrepo.SkillFilter{SourceName: "本机目录"}, 10, "")
	require.NoError(t, err)
	require.Len(t, skills, 2)
}
