package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"slices"
	"unicode/utf8"
)

type Location struct {
	SkillID, VersionID, LocalPath string
	GitHub                        *Source
}

type ChangeKind uint8

const (
	Added ChangeKind = iota + 1
	Modified
	Deleted
)

type FileChange struct {
	Path, LeftText, RightText, Detail string
	Kind                              ChangeKind
	Binary                            bool
}

type Comparison struct {
	Files                                            []FileChange
	LeftDigest, RightDigest, LeftCommit, RightCommit string
}

type comparisonSource struct {
	files          map[string]*zip.File
	digest, commit string
}

func (s *Library) comparisonSource(ctx context.Context, location Location) (comparisonSource, error) {
	var source comparisonSource
	count := 0
	for _, present := range []bool{location.LocalPath != "", location.GitHub != nil, location.SkillID != "", location.VersionID != ""} {
		if present {
			count++
		}
	}
	if count != 1 {
		return source, ErrInvalid
	}
	var reader *zip.Reader
	if location.LocalPath != "" || location.GitHub != nil {
		var dir string
		var err error
		if github := location.GitHub; github != nil {
			root, commit, cleanup, err := Checkout(ctx, github.Repository, github.Ref)
			if err != nil {
				return source, invalid(err)
			}
			defer cleanup()
			source.commit = commit
			dir, err = Within(root, github.Path)
			if err != nil {
				return source, invalid(err)
			}
		} else {
			dir, err = LocalDirectory(location.LocalPath)
			if err != nil {
				return source, err
			}
		}
		if _, err := ReadManifest(dir); err != nil {
			return source, invalid(err)
		}

		data, digest, err := Archive(dir)
		if err != nil {
			return source, invalid(err)
		}
		source.digest = digest
		reader, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return source, err
		}
	} else {
		skillID := location.SkillID
		id := location.VersionID
		if skillID != "" {
			skill, err := s.Skill(ctx, skillID)
			if err != nil {
				return source, err
			}
			id = skill.DefaultVersionId
		}

		row, err := s.Version(ctx, skillID, id)
		if err != nil {
			return source, err
		}

		data, err := s.ReadArchive(row)
		if err != nil {
			return source, err
		}
		source.digest, source.commit = row.ContentDigest, row.CommitSha
		reader, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return source, err
		}
	}
	source.files = make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		source.files[file.Name] = file
	}
	return source, nil
}

func readForComparison(file *zip.File) ([32]byte, string, bool, error) {
	if file == nil {
		return [32]byte{}, "", false, nil
	}

	r, err := file.Open()
	if err != nil {
		return [32]byte{}, "", false, err
	}
	defer r.Close()
	if file.UncompressedSize64 > PreviewLimit {
		hash := sha256.New()
		_, err := io.Copy(hash, r)
		return [32]byte(hash.Sum(nil)), "", true, err
	}
	data, err := io.ReadAll(r)
	binary := !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0
	if binary {
		return sha256.Sum256(data), "", true, err
	}
	return sha256.Sum256(data), string(data), false, err
}

func (s *Library) Compare(ctx context.Context, leftLocation, rightLocation Location) (*Comparison, error) {
	left, err := s.comparisonSource(ctx, leftLocation)
	if err != nil {
		return nil, err
	}

	right, err := s.comparisonSource(ctx, rightLocation)
	if err != nil {
		return nil, err
	}

	response := &Comparison{LeftDigest: left.digest, RightDigest: right.digest, LeftCommit: left.commit, RightCommit: right.commit}
	paths := make([]string, 0, len(left.files)+len(right.files))
	for path := range left.files {
		paths = append(paths, path)
	}
	for path := range right.files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	budget := 2 << 20
	for _, path := range slices.Compact(paths) {
		l, r := left.files[path], right.files[path]
		change := FileChange{Path: path, Kind: Modified}
		lDigest, leftText, leftBinary, err := readForComparison(l)
		if err != nil {
			return nil, err
		}

		rDigest, rightText, rightBinary, err := readForComparison(r)
		if err != nil {
			return nil, err
		}

		switch {
		case l == nil:
			change.Kind = Added
		case r == nil:
			change.Kind = Deleted
		case lDigest == rDigest && l.Mode() == r.Mode():
			continue
		}
		change.Binary = leftBinary || rightBinary
		size := len(leftText) + len(rightText)
		if change.Binary || size > budget {
			change.Detail = "文件无法预览或超过比对大小限制"
		} else {
			budget -= size
			change.LeftText, change.RightText = leftText, rightText
		}
		response.Files = append(response.Files, change)
	}
	return response, nil
}
