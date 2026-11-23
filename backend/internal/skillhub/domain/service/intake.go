package service

import (
	"cmp"
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func prepareImportMetadata(metadata Metadata) (Metadata, error) {
	tags, err := normalizeTags(metadata.Tags)
	if err != nil {
		return Metadata{}, err
	}
	metadata.Tags = tags
	metadata.Category = strings.TrimSpace(metadata.Category)
	metadata.Purpose = strings.TrimSpace(metadata.Purpose)
	metadata.UpstreamName = strings.TrimSpace(metadata.UpstreamName)
	metadata.UpstreamURL = strings.TrimSpace(metadata.UpstreamURL)
	if metadata.UpstreamURL != "" {
		link, err := url.Parse(metadata.UpstreamURL)
		if err != nil || link.Hostname() == "" || (link.Scheme != "https" && link.Scheme != "http") {
			return Metadata{}, invalid(errors.New("来源链接必须是有效的 HTTP 或 HTTPS 地址"))
		}
	}
	return metadata, nil
}

func (s *Library) ImportDirectories(ctx context.Context, repositoryURL, ref, sha string, paths []string, metadata Metadata) (*ImportResult, error) {
	metadata, err := prepareImportMetadata(metadata)
	if err != nil {
		return nil, err
	}
	source := Source{Kind: "folder", InitialMetadata: metadata}
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
		source.Kind, source.Repository, source.Ref, source.Commit = "github", repository, ref, commit
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

	s.commits.Lock()
	defer s.commits.Unlock()
	if err := s.cleanUploads(time.Now()); err != nil {
		return nil, err
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

const uploadLifetime = 24 * time.Hour

// Upload previews remain available for 24 hours, including failed import retries.
func (s *Library) cleanUploads(now time.Time) error {
	entries, err := os.ReadDir(s.uploads)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := uuid.Parse(entry.Name()); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if now.Sub(info.ModTime()) < uploadLifetime {
			continue
		}
		if err := os.RemoveAll(filepath.Join(s.uploads, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

type ArchiveImportOptions struct {
	MakeDefault     bool
	DirectoryUpload bool
	Metadata        Metadata
}

func (s *Library) ImportArchive(ctx context.Context, upload, target string, paths []string, options ArchiveImportOptions) (*ImportResult, error) {
	id, err := uuid.Parse(upload)
	if err != nil {
		return nil, ErrInvalid
	}
	if target != "" && len(paths) != 1 {
		return nil, ErrInvalid
	}
	metadata, err := prepareImportMetadata(options.Metadata)
	if err != nil {
		return nil, err
	}
	source := Source{Kind: "zip", InitialMetadata: metadata}
	if options.DirectoryUpload {
		source.Kind = "directory_upload"
	}
	s.commits.Lock()
	defer s.commits.Unlock()
	if err := s.cleanUploads(time.Now()); err != nil {
		return nil, err
	}
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
			_, created, err = s.saveDirectory(ctx, dir, target, source, VersionInput{MakeDefault: options.MakeDefault})
		}
		accountImport(&result, path, created, err)
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
