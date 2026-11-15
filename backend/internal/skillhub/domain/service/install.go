package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// InstallCommand produces a POSIX shell command; it does not execute it.
func InstallCommand(snapshotPath, repositoryURL, commit, name, projectPath string, global bool) (string, error) {
	if name == "" || len(name) > 64 || !validSkillName.MatchString(name) {
		return "", fmt.Errorf("invalid skill name")
	}
	var source string
	if snapshotPath != "" {
		if !filepath.IsAbs(snapshotPath) {
			return "", fmt.Errorf("snapshot path must be absolute")
		}
		info, err := os.Stat(snapshotPath)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("snapshot unavailable")
		}
		source = snapshotPath
	} else {
		u, err := url.Parse(repositoryURL)
		if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !repositoryPattern.MatchString(u.Path) || !commitPattern.MatchString(commit) {
			return "", fmt.Errorf("invalid pinned GitHub source")
		}
		source = strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git") + "@" + commit
	}
	command := "npx skills add " + shellQuote(source) + " --skill " + shellQuote(name) + " -a codex --copy"
	if global {
		return command + " -g", nil
	}
	if !filepath.IsAbs(projectPath) {
		return "", fmt.Errorf("project path must be absolute")
	}
	info, err := os.Stat(projectPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("project path must be an existing directory")
	}
	return "cd -- " + shellQuote(projectPath) + " && " + command, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
