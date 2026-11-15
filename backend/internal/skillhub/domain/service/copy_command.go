package service

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var repositorySkillPath = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// CopyCommand copies one stored snapshot or a pinned public Skill into the current directory.
func CopyCommand(snapshot, repository, repositoryPath, commit, name string) (string, error) {
	if len(name) > 64 || !validSkillName.MatchString(name) {
		return "", fmt.Errorf("invalid skill name")
	}
	target := shellQuote("./" + name)
	guard := "test ! -e " + target + " && test ! -L " + target + " && "
	if snapshot != "" {
		info, err := os.Stat(snapshot)
		if !filepath.IsAbs(snapshot) || err != nil || !info.IsDir() {
			return "", fmt.Errorf("snapshot unavailable")
		}
		return guard + "cp -R " + shellQuote(snapshot) + " " + target, nil
	}
	u, err := url.Parse(repository)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !repositoryPattern.MatchString(u.Path) || !commitPattern.MatchString(commit) {
		return "", fmt.Errorf("invalid pinned GitHub source")
	}
	subdir := path.Clean(repositoryPath)
	if repositoryPath != "" && (!repositorySkillPath.MatchString(repositoryPath) || strings.HasPrefix(repositoryPath, "/") || subdir == ".." || strings.HasPrefix(subdir, "../")) {
		return "", fmt.Errorf("invalid skill path")
	}
	source := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git")
	if subdir != "." {
		source += "/" + subdir
	}
	return guard + "npx --yes tiged@2.12.7 " + shellQuote(source+"#"+commit) + " " + target, nil
}
