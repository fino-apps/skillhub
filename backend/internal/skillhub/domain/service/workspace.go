package service

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const ArchiveLimit = 64 << 20
const PreviewLimit = 1 << 20

type Manifest struct {
	Name        string
	Description string
}

func ReadManifest(dir string) (Manifest, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Manifest{}, err
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open("SKILL.md")
	if err != nil {
		return Manifest{}, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, PreviewLimit+1))
	if err != nil {
		return Manifest{}, err
	}
	if len(data) > PreviewLimit {
		return Manifest{}, errors.New("SKILL.md 太大")
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return Manifest{}, errors.New("SKILL.md 缺少 YAML front matter")
	}
	header, _, ok := strings.Cut(text, "\n---")
	if !ok {
		return Manifest{}, errors.New("SKILL.md 缺少 YAML 结束标记")
	}
	var manifest Manifest
	if err := yaml.Unmarshal([]byte(header), &manifest); err != nil {
		return Manifest{}, err
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	if manifest.Name == "" {
		return Manifest{}, errors.New("SKILL.md 缺少 name")
	}
	return manifest, nil
}

type Candidate struct {
	Path string
	Manifest
}

func Scan(dir string) ([]Candidate, error) {
	var items []Candidate
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != dir && strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() || entry.Name() != "SKILL.md" {
			return nil
		}

		rel, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return err
		}
		manifest, err := ReadManifest(filepath.Dir(path))
		if err != nil {
			return nil
		}
		items = append(items, Candidate{filepath.ToSlash(rel), manifest})
		return nil
	})
	return items, err
}

func Files(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("不支持非常规文件 %s", entry.Name())
		}

		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		if len(paths) > 10000 {
			return errors.New("文件数量超过限制")
		}
		return nil
	})
	return paths, err
}

func Archive(dir string) ([]byte, string, error) {
	paths, err := Files(dir)
	if err != nil {
		return nil, "", err
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = root.Close() }()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	digest := sha256.New()
	var size int64
	for _, path := range paths {
		f, err := root.Open(path)
		if err != nil {
			return nil, "", err
		}

		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return nil, "", err
		}
		size += info.Size()
		if !info.Mode().IsRegular() || size > ArchiveLimit {
			_ = f.Close()
			return nil, "", errors.New("Skill 文件大小超过限制")
		}

		header := &zip.FileHeader{Name: path, Method: zip.Deflate}
		header.SetMode(0600 | info.Mode().Perm()&0111)
		dst, err := writer.CreateHeader(header)
		if err == nil {
			_, _ = fmt.Fprintf(digest, "%q:%d:%d\n", path, info.Size(), header.Mode().Perm())
			var copied int64
			copied, err = io.Copy(io.MultiWriter(dst, digest), io.LimitReader(f, info.Size()+1))
			if err == nil && copied != info.Size() {
				err = errors.New("收录过程中源文件发生变化")
			}
		}
		_ = f.Close()
		if err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return output.Bytes(), hex.EncodeToString(digest.Sum(nil)), nil
}

func Unpack(data []byte, dir string) error {
	if len(data) > ArchiveLimit {
		return errors.New("ZIP 大小超过限制")
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if len(reader.File) > 10000 {
		return errors.New("ZIP 文件数量超过限制")
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	var remaining int64 = ArchiveLimit
	for _, file := range reader.File {
		path := strings.TrimSuffix(file.Name, "/")
		if !fs.ValidPath(path) || path == "." || strings.ContainsAny(path, "\\:") {
			return errors.New("ZIP 包含无效路径")
		}

		if file.FileInfo().IsDir() {
			if err := root.MkdirAll(path, 0700); err != nil {
				return err
			}
			continue
		}
		if !file.Mode().IsRegular() || file.UncompressedSize64 > uint64(remaining) {
			return errors.New("ZIP 包含不支持的文件或大小超过限制")
		}
		if err := root.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}

		input, err := file.Open()
		if err != nil {
			return err
		}

		output, err := root.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600|file.Mode().Perm()&0111)
		if err != nil {
			_ = input.Close()
			return err
		}

		n, copyErr := io.Copy(output, io.LimitReader(input, remaining+1))
		err = errors.Join(copyErr, output.Close(), input.Close())
		if err != nil {
			return err
		}
		remaining -= n
		if remaining < 0 {
			return errors.New("ZIP 解压大小超过限制")
		}
	}
	return nil
}

func (s *Library) PickDirectory(ctx context.Context) (string, error) {
	if s.pickDirectory != nil {
		return s.pickDirectory(ctx)
	}
	return PickDirectory(ctx)
}

func PickDirectory(ctx context.Context) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "osascript", "-e", "POSIX path of (choose folder)")
	case "linux":
		cmd = exec.CommandContext(ctx, "zenity", "--file-selection", "--directory")
	default:
		return "", errors.New("当前平台不支持目录选择器")
	}

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func LocalDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", ErrInvalid
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", invalid(err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", invalid(err)
	}
	if !info.IsDir() {
		return "", invalid(errors.New("路径必须是目录"))
	}
	return resolved, nil
}

func SafeName(name string) string {
	return cmp.Or(strings.Trim(strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune("/\\:\"<>|?*", r) {
			return '-'
		}
		return r
	}, name), ". "), "skill")
}
