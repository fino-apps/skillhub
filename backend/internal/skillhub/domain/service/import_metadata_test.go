package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fino-io/finokit/db"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo"
	sqlite "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/model/sqlite"
	"github.com/stretchr/testify/require"
)

func TestImportMetadataAppliesOnlyToNewSkill(t *testing.T) {
	ctx := context.Background()
	provider, err := db.NewWithConfig(&db.Config{Driver: db.SqliteDriver, DSN: filepath.Join(t.TempDir(), "library.db")})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, provider.Close()) })
	require.NoError(t, provider.NewSession(ctx).AutoMigrate(&sqlite.Skill{}, &sqlite.SkillTag{}, &sqlite.SkillVersion{}))
	store := repo.NewLibrary(repo.LibraryDeps{DB: provider})
	library := NewLibrary(LibraryDeps{Store: store, SnapshotDir: t.TempDir()})

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample\n---\n"), 0600))
	source := Source{
		Kind: "github", Repository: "https://github.com/openai/skills",
		InitialMetadata: ImportMetadata{Category: "github.com/openai", Tags: []string{"docs"}},
	}
	version, created, err := library.saveDirectory(ctx, dir, "", source, VersionInput{})
	require.NoError(t, err)
	require.True(t, created)

	source.InitialMetadata = ImportMetadata{Category: "Other", Tags: []string{"other"}}
	_, created, err = library.saveDirectory(ctx, dir, "", source, VersionInput{})
	require.NoError(t, err)
	require.False(t, created)

	details, err := library.Detail(ctx, version.SkillId)
	require.NoError(t, err)
	require.Equal(t, "github.com/openai", details.Skill.Category)
	require.Equal(t, []string{"docs"}, details.Tags)
}
