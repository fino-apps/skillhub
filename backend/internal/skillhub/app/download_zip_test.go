package app

import (
	"archive/zip"
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	nhttp "github.com/fino-io/gokit/transport/http"
	"github.com/fino-io/skillhub/backend/internal/skillhub/api/endpoint"
	apihttp "github.com/fino-io/skillhub/backend/internal/skillhub/api/http"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type downloadStore struct {
	repo.Library
	version model.SkillVersionSnapshot
}

func (s downloadStore) Version(context.Context, string, string) (model.SkillVersionSnapshot, error) {
	return s.version, nil
}

func TestDownloadZipRootDirectory(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(source, "references"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(source, "references", "guide.md"), []byte("guide"), 0600))
	snapshot, _, err := service.Archive(source)
	require.NoError(t, err)

	skillID, versionID := uuid.NewString(), uuid.NewString()
	version := model.SkillVersionSnapshot{ID: model.SkillVersionID(versionID), SkillId: skillID, Name: "example-skill"}
	snapshots := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(snapshots, versionID+".zip"), snapshot, 0600))
	library := service.NewLibrary(service.LibraryDeps{Store: downloadStore{version: version}, SnapshotDir: snapshots})

	router := mux.NewRouter()
	apihttp.RegisterHttpHandler(router, endpoint.Endpoints{
		DownloadZipEndpoint: endpoint.MakeDownloadZipEndpoint(NewService(ServiceDeps{Library: library})),
	}, nhttp.NewConfig())
	path := "/skillhub/v1/skills/" + skillID + "/versions/" + versionID + "/archive"

	raw := httptest.NewRecorder()
	router.ServeHTTP(raw, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, raw.Code)
	require.Equal(t, snapshot, raw.Body.Bytes())

	download := httptest.NewRecorder()
	router.ServeHTTP(download, httptest.NewRequest(http.MethodGet, path+"?includeRootDirectory=true", nil))
	require.Equal(t, http.StatusOK, download.Code)
	require.Contains(t, download.Header().Get("Content-Disposition"), "example-skill.zip")
	archive, err := zip.NewReader(bytes.NewReader(download.Body.Bytes()), int64(download.Body.Len()))
	require.NoError(t, err)
	require.Len(t, archive.File, 2)
	for path, want := range map[string]string{
		"example-skill/SKILL.md":            "skill",
		"example-skill/references/guide.md": "guide",
	} {
		content, err := fs.ReadFile(archive, path)
		require.NoError(t, err)
		require.Equal(t, want, string(content))
	}
}
