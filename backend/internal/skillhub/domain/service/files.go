package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// skillFile describes a file used for comparison.
type skillFile struct {
	Path   string
	Size   int64
	Digest string
	Mode   fs.FileMode
}

// readSkillFiles returns a stable digest and file list without following links.
func readSkillFiles(path string) ([]skillFile, string, error) {
	return inspectSkillFiles(path, "")
}

func inspectSkillFiles(path, destination string) ([]skillFile, string, error) {
	if !filepath.IsAbs(path) {
		return nil, "", fmt.Errorf("skill path must be absolute")
	}
	root, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, "", fmt.Errorf("skill path must be a directory")
	}
	if _, ok, err := readSkill(root); err != nil || !ok {
		return nil, "", fmt.Errorf("invalid SKILL.md")
	}
	source, err := os.OpenRoot(root)
	if err != nil {
		return nil, "", err
	}
	defer source.Close()
	files := make([]skillFile, 0)
	var total int64
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not supported: %s", rel)
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == ".venv" {
				return filepath.SkipDir
			}
			if destination != "" {
				return os.MkdirAll(filepath.Join(destination, rel), 0700)
			}
			return nil
		}
		if entry.Name() == ".git" {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported file: %s", rel)
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		if stat.Size() > 20*1024*1024 {
			return fmt.Errorf("file too large: %s", rel)
		}
		total += stat.Size()
		if total > 64*1024*1024 || len(files) >= 10000 {
			return fmt.Errorf("skill exceeds snapshot size or file count limit")
		}
		target := ""
		if destination != "" {
			target = filepath.Join(destination, rel)
		}
		digest, err := hashSkillFile(source, rel, stat, target)
		if err != nil {
			return err
		}
		files = append(files, skillFile{Path: filepath.ToSlash(rel), Size: stat.Size(), Digest: digest, Mode: stat.Mode().Perm() & 0100})
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	hash := sha256.New()
	for _, file := range files {
		fmt.Fprintf(hash, "%s\x00%s\x00%o\x00", file.Path, file.Digest, file.Mode)
	}
	return files, hex.EncodeToString(hash.Sum(nil)), nil
}

// hashSkillFile hashes the bytes written to a snapshot or read for comparison.
func hashSkillFile(source *os.Root, relative string, info fs.FileInfo, destination string) (string, error) {
	file, err := source.Open(relative)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	var writer io.Writer = hash
	var target *os.File
	if destination != "" {
		target, err = os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600|info.Mode().Perm()&0100)
		if err != nil {
			return "", err
		}
		writer = io.MultiWriter(hash, target)
	}
	size, err := io.Copy(writer, io.LimitReader(file, info.Size()+1))
	if target != nil {
		closeErr := target.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return "", err
	}
	if size != info.Size() {
		return "", fmt.Errorf("skill changed while reading %s", relative)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
