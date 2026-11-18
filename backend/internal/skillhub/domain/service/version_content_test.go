package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

type installStore struct {
	repo.Library
	version model.SkillVersionSnapshot
}

func (s installStore) Version(context.Context, string, string) (model.SkillVersionSnapshot, error) {
	return s.version, nil
}

func TestInstallCommandDownloadsArchive(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl unavailable")
	}
	if _, err := exec.LookPath("unzip"); err != nil {
		t.Skip("unzip unavailable")
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("installed"), 0600); err != nil {
		t.Fatal(err)
	}
	data, _, err := Archive(source)
	if err != nil {
		t.Fatal(err)
	}
	skill, version := uuid.NewString(), uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/skillhub/v1/skills/"+skill+"/versions/"+version+"/archive" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	snapshots := t.TempDir()
	if err := os.WriteFile(filepath.Join(snapshots, version+".zip"), data, 0600); err != nil {
		t.Fatal(err)
	}
	row := model.SkillVersionSnapshot{ID: model.SkillVersionID(version), SkillId: skill, Name: "sample's skill"}
	library := NewLibrary(LibraryDeps{Store: installStore{version: row}, SnapshotDir: snapshots})
	for _, global := range []bool{false, true} {
		work := t.TempDir()
		home := t.TempDir()
		command, err := library.InstallCommand(context.Background(), skill, version, global)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = work
		cmd.Env = append(os.Environ(), "SKILLHUB_URL="+server.URL, "HOME="+home)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("install global=%v: %v\n%s", global, err, output)
		}
		target := work
		if global {
			target = home
		}
		content, err := os.ReadFile(filepath.Join(target, ".agents", "skills", SafeName(row.Name), "SKILL.md"))
		if err != nil || string(content) != "installed" {
			t.Fatalf("installed content = %q, %v", content, err)
		}
	}
}
