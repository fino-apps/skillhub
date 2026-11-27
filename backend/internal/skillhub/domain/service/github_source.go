package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var repositoryPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var CommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func GitHubURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("需要 GitHub HTTPS 仓库地址")
	}

	owner, name, ok := strings.Cut(strings.Trim(u.Path, "/"), "/")
	name = strings.TrimSuffix(name, ".git")
	for _, part := range []string{owner, name} {
		if !ok || !repositoryPart.MatchString(part) || part == "." || part == ".." {
			return "", errors.New("无效的 GitHub 仓库地址")
		}
	}
	return "https://github.com/" + strings.ToLower(owner+"/"+name), nil
}

func Checkout(ctx context.Context, raw, ref string) (string, string, func(), error) {
	repository, err := GitHubURL(raw)
	if err != nil {
		return "", "", nil, err
	}
	if ref == "" {
		ref = "HEAD"
	}
	if strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, ": \t\r\n") {
		return "", "", nil, errors.New("无效的 Git 引用")
	}

	dir, err := os.MkdirTemp("", "skillhub-git-")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	}

	for _, args := range [][]string{{"init", "-q"}, {"-c", "protocol.file.allow=never", "fetch", "-q", "--depth=1", "--", repository, ref}, {"checkout", "-q", "--detach", "FETCH_HEAD"}} {
		if _, err := run(args...); err != nil {
			cleanup()
			return "", "", nil, err
		}
	}

	commit, err := run("rev-parse", "HEAD")
	if err != nil {
		cleanup()
		return "", "", nil, err
	}
	return dir, commit, cleanup, nil
}

func Within(dir, path string) (string, error) {
	if path == "" {
		path = "."
	}
	if !fs.ValidPath(path) || strings.ContainsAny(path, "\\:") {
		return "", errors.New("无效的 Skill 路径")
	}

	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer func() { _ = root.Close() }()
	info, err := root.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("Skill 路径必须是目录")
	}

	full := filepath.Join(dir, filepath.FromSlash(path))
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(dir, resolved)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errors.New("Skill 路径超出根目录")
	}
	return resolved, nil
}
