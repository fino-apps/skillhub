package service

import (
	"cmp"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

type ImportFailure struct {
	Path string
	Err  error
}

type ImportResult struct {
	Added, Existing int
	Failures        []ImportFailure
}

type Preview struct {
	Commit string
	Items  []Candidate
}

type Upload struct {
	ID    string
	Items []Candidate
}

func PreviewGitHub(ctx context.Context, source Source, collection bool) (*Preview, error) {
	root, commit, clean, err := Checkout(ctx, source.Repository, source.Ref)
	if err != nil {
		return nil, invalid(err)
	}
	defer clean()
	path := source.Path
	if collection && path == "" {
		path = "skills"
	}

	dir, err := Within(root, path)
	if err != nil {
		return nil, invalid(err)
	}
	if !collection {
		if manifest, err := ReadManifest(dir); err == nil {
			return &Preview{Commit: commit, Items: []Candidate{{Path: path, Manifest: manifest}}}, nil
		}
	}

	items, err := Scan(dir)
	if err != nil {
		return nil, invalid(err)
	}
	for i := range items {
		items[i].Path = filepath.ToSlash(filepath.Join(path, items[i].Path))
	}
	return &Preview{Commit: commit, Items: items}, nil
}

func (s *Library) ImportDirectories(ctx context.Context, repositoryURL, ref, sha string, paths []string) (*ImportResult, error) {
	source := Source{Kind: "folder"}
	var root string
	if repositoryURL != "" {
		repository, err := GitHubURL(repositoryURL)
		if err != nil || !CommitPattern.MatchString(sha) {
			return nil, ErrInvalid
		}

		dir, commit, cleanup, err := Checkout(ctx, repository, sha)
		if err != nil {
			return nil, invalid(err)
		}
		defer cleanup()
		root = dir
		source = Source{Kind: "github", Repository: repository, Ref: ref, Commit: commit}
	}
	var result ImportResult
	for _, path := range paths {
		var dir string
		var err error
		if root == "" {
			dir, err = LocalDirectory(path)
		} else {
			dir, err = Within(root, path)
			source.Path = filepath.ToSlash(filepath.Clean(path))
			if source.Path == "." {
				source.Path = ""
			}
		}

		created := false
		if err == nil {
			_, created, err = s.saveDirectory(ctx, dir, "", source, VersionInput{})
		}
		accountImport(&result, path, created, err)
	}
	return &result, nil
}

func accountImport(result *ImportResult, path string, created bool, err error) {
	switch {
	case err != nil:
		result.Failures = append(result.Failures, ImportFailure{Path: path, Err: err})
	case created:
		result.Added++
	default:
		result.Existing++
	}
}

func (s *Library) StageArchive(file []byte) (*Upload, error) {
	if len(file) == 0 {
		return nil, ErrInvalid
	}

	id := uuid.NewString()
	dir := filepath.Join(s.uploads, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		return nil, err
	}
	valid := false
	defer func() {
		if !valid {
			_ = os.RemoveAll(dir)
		}
	}()
	if err := Unpack(file, dir); err != nil {
		return nil, invalid(err)
	}

	items, err := Scan(dir)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrInvalid
	}
	valid = true
	return &Upload{ID: id, Items: items}, nil
}

func (s *Library) ImportArchive(ctx context.Context, upload, target string, paths []string, makeDefault bool) (*ImportResult, error) {
	id, err := uuid.Parse(upload)
	if err != nil {
		return nil, ErrInvalid
	}
	if target != "" && len(paths) != 1 {
		return nil, ErrInvalid
	}
	s.commits.Lock()
	defer s.commits.Unlock()
	root := filepath.Join(s.uploads, id.String())
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, repo.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	var result ImportResult
	for _, path := range paths {
		dir, err := Within(root, path)
		created := false
		if err == nil {
			_, created, err = s.saveDirectory(ctx, dir, target, Source{Kind: "zip"}, VersionInput{MakeDefault: makeDefault})
		}
		accountImport(&result, path, created, err)
	}

	if len(result.Failures) == 0 {
		_ = os.RemoveAll(root)
	}
	return &result, nil
}

func (s *Library) CreateVersion(ctx context.Context, skillID, path, ref, commitSHA string, input VersionInput) (model.SkillVersionSnapshot, error) {
	skill, err := s.Skill(ctx, skillID)
	if err != nil {
		return model.SkillVersionSnapshot{}, err
	}

	source := Source{Kind: "folder"}
	if path == "" && skill.RepositoryUrl != "" {
		if !CommitPattern.MatchString(commitSHA) {
			return model.SkillVersionSnapshot{}, ErrInvalid
		}
		root, commit, cleanup, err := Checkout(ctx, skill.RepositoryUrl, commitSHA)
		if err != nil {
			return model.SkillVersionSnapshot{}, invalid(err)
		}
		defer cleanup()
		path, err = Within(root, skill.RepositoryPath)
		if err != nil {
			return model.SkillVersionSnapshot{}, invalid(err)
		}
		source = Source{Kind: "github", Repository: skill.RepositoryUrl, Path: skill.RepositoryPath, Ref: ref, Commit: commit}
	} else {
		path, err = LocalDirectory(strings.TrimSpace(cmp.Or(path, skill.LocalPath)))
		if err != nil {
			return model.SkillVersionSnapshot{}, err
		}
	}

	version, _, err := s.saveDirectory(ctx, path, skillID, source, input)
	if err != nil {
		return model.SkillVersionSnapshot{}, err
	}
	return version, nil
}
