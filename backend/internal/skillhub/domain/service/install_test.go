package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCommand(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project 'one'")
	snapshot := filepath.Join(root, "my snapshot")
	for _, path := range []string{project, snapshot} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	command, err := InstallCommand(snapshot, "", "", "my-skill", project, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, "project '\\''one'\\''") || !strings.Contains(command, "'my snapshot'") && !strings.Contains(command, "my snapshot'") || !strings.Contains(command, "-a codex --copy") {
		t.Fatalf("unexpected command: %s", command)
	}
	commit := strings.Repeat("a", 40)
	command, err = InstallCommand("", "https://github.com/example/repo", commit, "my-skill", "", true)
	if err != nil || !strings.Contains(command, "'example/repo@"+commit+"'") || !strings.HasSuffix(command, " -g") {
		t.Fatalf("unexpected command: %s, %v", command, err)
	}
}

func TestInstallCommandRejectsInvalidInput(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		snapshot, repo, commit, name, project string
		global                                bool
	}{
		{"", "https://example.com/org/repo", strings.Repeat("a", 40), "skill", "", true},
		{"", "https://github.com/org/repo", "main", "skill", "", true},
		{root, "", "", "skill;bad", "", true},
		{root, "", "", "skill", "relative/path", false},
	} {
		if _, err := InstallCommand(tc.snapshot, tc.repo, tc.commit, tc.name, tc.project, tc.global); err == nil {
			t.Fatalf("expected invalid input to be rejected: %+v", tc)
		}
	}
}
