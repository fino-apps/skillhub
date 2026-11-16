package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/fino-io/skillhub/go/skillhub/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestDesktopCatalogWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILLHUB_DATA_DIR", filepath.Join(dir, "data"))
	t.Setenv("SKILLHUB_DATABASE", "")
	t.Setenv("SKILLHUB_SNAPSHOT_DIR", "")
	t.Setenv("SKILLHUB_EXPORT_DIR", filepath.Join(dir, "exports"))
	t.Setenv("SKILLHUB_OLLAMA_MODEL", "")
	_, err := loadConfig()
	require.NoError(t, err)
	handler, close, err := newDesktop(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, close()) })

	request := func(method, path string, body io.Reader, contentType string, status int) []byte {
		t.Helper()
		r := httptest.NewRequest(method, "http://127.0.0.1"+path, body)
		r.Header.Set("Content-Type", contentType)
		reply := httptest.NewRecorder()
		handler.ServeHTTP(reply, r)
		require.Equal(t, status, reply.Code, "%s %s: %s", method, path, reply.Body)
		return reply.Body.Bytes()
	}
	api := func(method, path string, body, result any) {
		t.Helper()
		var input io.Reader = http.NoBody
		if body != nil {
			data, err := json.Marshal(body)
			require.NoError(t, err)
			input = bytes.NewReader(data)
		}
		data := request(method, "/skillhub/v1"+path, input, "application/json", http.StatusOK)
		if result != nil {
			if message, ok := result.(proto.Message); ok {
				proto.Reset(message)
			}
			require.NoError(t, json.Unmarshal(data, result), "%s", data)
		}
	}

	skillDir := filepath.Join(dir, "sample")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "references"), 0700))
	write := func(name, text string, mode os.FileMode) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, name), []byte(text), mode))
	}
	write("SKILL.md", "---\nname: sample\ndescription: Original description\n---\n\n[Guide](references/guide.md)\n", 0600)
	write("references/guide.md", "# First\n", 0600)
	write("run.sh", "#!/bin/sh\n", 0700)
	alias := filepath.Join(dir, "alias")
	require.NoError(t, os.Symlink(skillDir, alias))

	var scan pb.ScanProjectResponse
	api("POST", "/projects:scan", map[string]any{"projectPath": alias}, &scan)
	require.Len(t, scan.Skills, 1)
	var imported pb.ImportSkillResponse
	api("POST", "/imports", map[string]any{"skillPaths": []string{alias}}, &imported)
	require.EqualValues(t, 1, imported.AddedCount)
	api("POST", "/imports", map[string]any{"skillPaths": []string{skillDir}}, &imported)
	require.EqualValues(t, 1, imported.ExistingCount)

	var library pb.ListSkillsResponse
	api("GET", "/skills?keyword=original", nil, &library)
	require.Len(t, library.Skills, 1)
	skill := library.Skills[0]
	first := skill.GetDefaultVersion()
	require.True(t, first.GetAvailable())
	resource := "/skills/" + skill.Id
	versionPath := resource + "/versions/" + first.Id

	var content pb.GetVersionContentResponse
	api("GET", versionPath+"/content", nil, &content)
	require.Contains(t, content.Files, "references/guide.md")
	var file pb.GetVersionFileResponse
	api("GET", versionPath+"/file?path="+url.QueryEscape("references/guide.md"), nil, &file)
	require.Equal(t, "# First\n", file.Text)
	request("GET", "/skillhub/v1"+versionPath+"/file?path="+url.QueryEscape("../outside"), http.NoBody, "", http.StatusBadRequest)

	write("references/guide.md", "# Second\n", 0600)
	var updated pb.CreateVersionResponse
	api("POST", resource+"/versions", map[string]any{"localPath": skillDir}, &updated)
	second := updated.GetVersion()
	require.NotEqual(t, first.Id, second.Id)
	var detail pb.GetSkillResponse
	api("GET", resource, nil, &detail)
	require.Equal(t, first.Id, detail.Skill.GetDefaultVersion().GetId())

	var comparison struct {
		Files []struct{ Path, RightText string }
	}
	api("POST", "/skills:compare", map[string]any{
		"left": map[string]string{"versionId": first.Id}, "right": map[string]string{"versionId": second.Id},
	}, &comparison)
	require.Len(t, comparison.Files, 1)
	require.Equal(t, "references/guide.md", comparison.Files[0].Path)
	require.Equal(t, "# Second\n", comparison.Files[0].RightText)

	api("PUT", resource+"/versions/"+second.Id+"/default", nil, nil)
	api("GET", resource, nil, &detail)
	require.Equal(t, second.Id, detail.Skill.GetDefaultVersion().GetId())
	api("PATCH", "/skills:organize", map[string]any{"changes": []map[string]any{{
		"skillId": skill.Id, "purpose": "短用途", "category": "开发", "addTags": []string{"Go", "Shell"},
	}}}, nil)
	api("GET", "/skills?keyword="+url.QueryEscape("短用途")+"&tags=Go&tags=Shell", nil, &library)
	require.Len(t, library.Skills, 1)
	api("GET", "/skills?tags=Go&tags=Other", nil, &library)
	require.Empty(t, library.Skills)
	var facets pb.ListFacetsResponse
	api("GET", "/facets", nil, &facets)
	require.Equal(t, []string{"folder"}, facets.Sources)
	require.Equal(t, []string{"Go", "Shell"}, facets.Tags)

	var exported pb.ExportVersionToDirectoryResponse
	api("POST", versionPath+":export", map[string]any{}, &exported)
	expected, err := filepath.EvalSymlinks(filepath.Join(dir, "exports", "sample"))
	require.NoError(t, err)
	require.Equal(t, expected, exported.Path)
	info, err := os.Stat(filepath.Join(exported.Path, "run.sh"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0100)

	archive := request("GET", "/skillhub/v1"+versionPath+"/archive", http.NoBody, "", http.StatusOK)
	require.NotEmpty(t, archive)
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)
	require.Len(t, reader.File, 5)
	var form bytes.Buffer
	multipartWriter := multipart.NewWriter(&form)
	part, err := multipartWriter.CreateFormFile("file", "sample.zip")
	require.NoError(t, err)
	_, err = part.Write(archive)
	require.NoError(t, err)
	require.NoError(t, multipartWriter.Close())
	preview := request("POST", "/skillhub/v1/uploads", &form, multipartWriter.FormDataContentType(), http.StatusOK)
	var upload struct{ UploadID string }
	require.NoError(t, json.Unmarshal(preview, &upload))
	api("POST", "/uploads/"+upload.UploadID+":commit", map[string]any{"skillPaths": []string{"sample"}}, &imported)
	require.EqualValues(t, 1, imported.AddedCount, "%+v", imported.Failures)
	api("DELETE", versionPath, nil, nil)
	api("DELETE", resource, nil, nil)
}
