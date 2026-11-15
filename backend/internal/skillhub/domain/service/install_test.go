package service

import (
	"strings"
	"testing"
)

func TestInstallCommandPinsCommitAndDirectory(t *testing.T) {
	commit := strings.Repeat("a", 40)
	for _, tc := range []struct {
		path, source string
		global       bool
	}{
		{"", "https://github.com/example/repo#" + commit, false},
		{"skills/playwright", "https://github.com/example/repo/tree/" + commit + "/skills/playwright", true},
	} {
		command, err := InstallCommand("https://github.com/Example/Repo.git", tc.path, commit, "playwright", tc.global)
		want := "npx skills@1.8.1 add '" + tc.source + "' --skill 'playwright' -a codex --copy"
		if tc.global {
			want += " -g"
		}
		if err != nil || command != want {
			t.Fatalf("InstallCommand(%q) = %q, %v, want %q", tc.path, command, err, want)
		}
	}
	if _, err := InstallCommand("https://github.com/example/repo", "../playwright", commit, "playwright", false); err == nil {
		t.Fatal("accepted a path outside the repository")
	}
	if _, err := InstallCommand("https://github.com/example/repo", "", "main", "playwright", false); err == nil {
		t.Fatal("accepted a mutable Git ref")
	}
}
