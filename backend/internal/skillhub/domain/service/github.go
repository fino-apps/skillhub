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

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	skillpb "github.com/fino-io/skillhub/go/skillhub"
)

type GitHubSource struct{ RepositoryURL, SkillPath, Ref string }
type GitHubPreview struct {
	Files                             []ImportFile
	Digest, Commit, Name, Description string
}

var repositoryPattern = regexp.MustCompile(`^/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:\.git)?$`)

// CheckoutGitHub resolves a public repository into a temporary directory.
// The caller must invoke cleanup after use; no repository code is executed.
func CheckoutGitHub(ctx context.Context, source GitHubSource) (root, commit string, cleanup func(), err error) {
	u, err := url.Parse(source.RepositoryURL)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !repositoryPattern.MatchString(u.Path) {
		return "", "", nil, fmt.Errorf("invalid GitHub repository URL")
	}
	ref := strings.TrimSpace(source.Ref)
	if ref == "" || len(ref) > 200 || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, " \t\r\n\\~^:?*[") || strings.Contains(ref, "..") {
		return "", "", nil, fmt.Errorf("invalid git reference")
	}
	path := filepath.Clean(source.SkillPath)
	if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", "", nil, fmt.Errorf("invalid skill path")
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
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("GitHub fetch failed: %w", err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err = run("init", "-q", temp); err != nil {
		return "", "", nil, err
	}
	if _, err = run("-C", temp, "-c", "protocol.file.allow=never", "fetch", "-q", "--depth=1", source.RepositoryURL, ref); err != nil {
		return "", "", nil, err
	}
	commit, err = run("-C", temp, "rev-parse", "FETCH_HEAD^{commit}")
	if err != nil {
		return "", "", nil, err
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

// PreviewGitHub inspects a pinned repository without executing its files.
func PreviewGitHub(ctx context.Context, source GitHubSource) (GitHubPreview, error) {
	root, commit, cleanup, err := CheckoutGitHub(ctx, source)
	if err != nil {
		return GitHubPreview{}, err
	}
	defer cleanup()
	meta, ok, err := readSkill(root)
	if err != nil || !ok || (len(meta.Name) > 64 || !validSkillName.MatchString(meta.Name)) {
		return GitHubPreview{}, fmt.Errorf("invalid SKILL.md")
	}
	files, digest, err := PreviewLocalSkill(root)
	if err != nil {
		return GitHubPreview{}, err
	}
	return GitHubPreview{Files: files, Digest: digest, Commit: commit, Name: meta.Name, Description: meta.Description}, nil
}

// ImportPublic saves only a resolved upstream reference and version metadata.
func (i *Intake) ImportPublic(ctx context.Context, source GitHubSource, expected, commit, version, notes string) (*model.Skill, *model.SkillVersion, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	preview, err := PreviewGitHub(ctx, source)
	if err != nil {
		return nil, nil, err
	}
	source.RepositoryURL = strings.TrimSuffix(strings.ToLower(source.RepositoryURL), ".git")
	source.SkillPath = filepath.ToSlash(filepath.Clean(source.SkillPath))
	if source.SkillPath == "." {
		source.SkillPath = ""
	}
	if preview.Digest != expected || preview.Commit != commit {
		return nil, nil, fmt.Errorf("source changed since preview")
	}
	skill, err := model.NewSkill(model.CreateSkillInput{Name: preview.Name, Description: preview.Description, Kind: skillpb.SkillKind_SKILL_KIND_PUBLIC, RepositoryUrl: source.RepositoryURL, RepositoryPath: source.SkillPath})
	if err != nil {
		return nil, nil, err
	}
	if version == "" {
		version = "1.0.0"
	}
	initial, err := model.NewSkillVersion(model.CreateSkillVersionInput{SkillId: string(skill.ID()), Version: version, Notes: notes, Digest: preview.Digest, SourceRef: source.Ref, CommitSha: preview.Commit})
	if err != nil {
		return nil, nil, err
	}
	if err := i.repository.CreateInitial(ctx, skill, initial, func() error { return nil }); err != nil {
		return nil, nil, err
	}
	return skill, initial, nil
}
