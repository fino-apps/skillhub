package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type GitHubSource struct{ RepositoryURL, SkillPath, Ref string }

var repositoryPattern = regexp.MustCompile(`^/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:\.git)?$`)

var validSkillName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var skillPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

func NormalizeRepository(repository string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(repository))
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" ||
		!repositoryPattern.MatchString(u.Path) {
		return "", fmt.Errorf("invalid GitHub repository URL")
	}
	return "https://github.com" + strings.TrimSuffix(strings.ToLower(u.Path), ".git"), nil
}

func repositoryPath(path string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." && (path == "" || path == ".") {
		return "", nil
	}
	if len(path) > 1024 || !skillPathPattern.MatchString(path) ||
		clean != path || strings.HasPrefix(path, "/") || path == ".." ||
		strings.HasPrefix(path, "../") || strings.Contains("/"+path+"/", "/.git/") {
		return "", fmt.Errorf("invalid skill path")
	}
	return path, nil
}

func validGitRef(ref string) bool {
	return ref != "" && len(ref) <= 200 && !strings.HasPrefix(ref, "-") &&
		!strings.ContainsAny(ref, " \t\r\n\\~^:?*[") && !strings.Contains(ref, "..")
}

// CheckoutGitHub checks out the requested commit without executing repository content.
func CheckoutGitHub(ctx context.Context, source GitHubSource) (root, commit string, cleanup func(), err error) {
	repository, err := NormalizeRepository(source.RepositoryURL)
	if err != nil {
		return "", "", nil, err
	}
	ref := strings.TrimSpace(source.Ref)
	if ref == "" {
		ref = "HEAD"
	}
	if !validGitRef(ref) {
		return "", "", nil, fmt.Errorf("invalid git reference")
	}
	path, err := repositoryPath(source.SkillPath)
	if err != nil {
		return "", "", nil, err
	}
	temp, err := os.MkdirTemp("", "skillhub-github-")
	if err != nil {
		return "", "", nil, err
	}
	created := temp
	remove := func() { _ = os.RemoveAll(created) }
	cleanup = remove
	defer func() {
		if err != nil {
			remove()
		}
	}()
	temp, err = filepath.EvalSymlinks(temp)
	if err != nil {
		return "", "", nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("GitHub fetch failed: %w", err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err = run("init", "-q", temp); err != nil {
		return "", "", nil, err
	}
	if _, err = run("-C", temp, "-c", "protocol.file.allow=never", "fetch", "-q", "--depth=1", repository, ref); err != nil {
		return "", "", nil, err
	}
	commit, err = run("-C", temp, "rev-parse", "FETCH_HEAD^{commit}")
	if err != nil {
		return "", "", nil, err
	}
	if commitPattern.MatchString(ref) && commit != ref {
		return "", "", nil, fmt.Errorf("resolved commit does not match selected commit")
	}
	if _, err = run("-C", temp, "-c", "core.hooksPath=/dev/null", "checkout", "-q", "--detach", commit); err != nil {
		return "", "", nil, err
	}
	root, err = filepath.EvalSymlinks(filepath.Join(temp, path))
	if err != nil || !within(temp, root) {
		return "", "", nil, fmt.Errorf("skill path is outside repository or unavailable")
	}
	return root, commit, cleanup, nil
}

// InspectGitHub reads one Skill or discovers the direct children of skills/.
func InspectGitHub(ctx context.Context, source GitHubSource, collection bool) ([]ScannedSkill, string, error) {
	root, commit, cleanup, err := CheckoutGitHub(ctx, GitHubSource{RepositoryURL: source.RepositoryURL, Ref: source.Ref})
	if err != nil {
		return nil, "", err
	}
	defer cleanup()

	directory, err := repositoryPath(source.SkillPath)
	if err != nil {
		return nil, "", err
	}
	if !collection {
		if skill, err := ReadGitHubSkill(root, directory); err == nil {
			return []ScannedSkill{skill}, commit, nil
		}
	}
	if directory == "" {
		directory = "skills"
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, directory))
	if os.IsNotExist(err) {
		return nil, commit, nil
	}
	if err != nil || resolved != filepath.Join(root, directory) {
		return nil, "", fmt.Errorf("invalid skills directory")
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, "", err
	}
	skills := make([]ScannedSkill, 0)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if skill, err := ReadGitHubSkill(root, directory+"/"+entry.Name()); err == nil {
			skills = append(skills, skill)
		}
	}
	return skills, commit, nil
}

func ReadGitHubSkill(root, path string) (ScannedSkill, error) {
	path, err := repositoryPath(path)
	if err != nil {
		return ScannedSkill{}, err
	}
	directory := filepath.Join(root, path)
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		return ScannedSkill{}, fmt.Errorf("invalid skill directory: %s", path)
	}
	skill, ok, err := readSkill(directory)
	if err != nil {
		return ScannedSkill{}, err
	}
	if !ok || len(skill.Name) > 64 || !validSkillName.MatchString(skill.Name) || len(skill.Description) > 1024 {
		return ScannedSkill{}, fmt.Errorf("invalid SKILL.md: %s", path)
	}
	skill.Path = path
	return skill, nil
}

func IsCommit(value string) bool {
	return commitPattern.MatchString(value)
}
