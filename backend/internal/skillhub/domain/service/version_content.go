package service

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
)

type VersionContents struct {
	Description string
	Files       []string
}

type FilePreview struct {
	Text             string
	Size             int32
	Binary, TooLarge bool
}

func (s *Library) openVersion(ctx context.Context, skill, id string) (model.SkillVersionSnapshot, *zip.Reader, error) {
	row, err := s.Version(ctx, skill, id)
	if err != nil {
		return row, nil, err
	}

	data, err := s.ReadArchive(row)
	if err != nil {
		return row, nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	return row, reader, err
}

func (s *Library) VersionContents(ctx context.Context, skill, id string) (VersionContents, error) {
	row, archive, err := s.openVersion(ctx, skill, id)
	if err != nil {
		return VersionContents{}, err
	}

	response := VersionContents{Description: row.Description}
	for _, file := range archive.File {
		if !file.FileInfo().IsDir() {
			response.Files = append(response.Files, file.Name)
		}
	}
	return response, nil
}

func (s *Library) VersionFile(ctx context.Context, skill, id, path string) (FilePreview, error) {
	_, archive, err := s.openVersion(ctx, skill, id)
	if err != nil {
		return FilePreview{}, err
	}

	for _, file := range archive.File {
		if file.Name != path || file.FileInfo().IsDir() {
			continue
		}

		result := FilePreview{Size: int32(file.UncompressedSize64)}
		if file.UncompressedSize64 > PreviewLimit {
			result.TooLarge = true
			return result, nil
		}

		r, err := file.Open()
		if err != nil {
			return FilePreview{}, err
		}
		defer r.Close()
		data, err := io.ReadAll(io.LimitReader(r, PreviewLimit+1))
		if err != nil {
			return FilePreview{}, err
		}
		result.Binary = !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0
		if !result.Binary {
			result.Text = string(data)
		}
		return result, nil
	}
	return FilePreview{}, repo.ErrNotFound
}

func (s *Library) DownloadArchive(ctx context.Context, skill, id string, includeRootDirectory bool) (string, []byte, error) {
	row, err := s.Version(ctx, skill, id)
	if err != nil {
		return "", nil, err
	}

	name := SafeName(row.Name)
	data, err := s.ReadArchive(row)
	if err != nil || !includeRootDirectory {
		return name, data, err
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", nil, err
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range archive.File {
		entry := *file
		entry.Name = name + "/" + file.Name
		if err := writer.Copy(&entry); err != nil {
			return "", nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return "", nil, err
	}
	return name, output.Bytes(), nil
}

func (s *Library) Export(ctx context.Context, skill, version, base string, choose bool) (string, error) {
	row, err := s.Version(ctx, skill, version)
	if err != nil {
		return "", err
	}

	data, err := s.ReadArchive(row)
	if err != nil {
		return "", err
	}

	if choose {
		base, err = PickDirectory(ctx)
		if err != nil {
			return "", invalid(err)
		}
	}
	base = cmp.Or(base, s.exports)
	if !filepath.IsAbs(base) {
		return "", ErrInvalid
	}

	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}

	dir := filepath.Join(base, SafeName(row.Name))
	if err := os.Mkdir(dir, 0700); errors.Is(err, os.ErrExist) {
		dir, err = os.MkdirTemp(base, SafeName(row.Name)+"-")
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	if err := Unpack(data, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func (s *Library) InstallCommand(ctx context.Context, skill, version string, global bool) (string, error) {
	row, err := s.Version(ctx, skill, version)
	if err != nil {
		return "", err
	}
	if !s.Available(row) {
		return "", repo.ErrNotFound
	}

	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	target := quote("./.agents/skills/" + SafeName(row.Name))
	if global {
		target = "\"$HOME\"/" + quote(".agents/skills/"+SafeName(row.Name))
	}
	archivePath := "/skillhub/v1/skills/" + skill + "/versions/" + version + "/archive"
	command := "( archive=$(mktemp) && trap 'rm -f \"$archive\"' EXIT && curl -fSL -o \"$archive\" -- \"${SKILLHUB_URL:?Set SKILLHUB_URL to the SkillHub address}" + archivePath + "\" && mkdir -p " + target + " && unzip -o \"$archive\" -d " + target + " )"
	return command, nil
}
