package service

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ArchiveOrigin struct {
	Name       string `json:"name,omitempty"`
	URL        string `json:"url,omitempty"`
	Repository string `json:"repository,omitempty"`
	SkillPath  string `json:"skillPath,omitempty"`
	Commit     string `json:"commit,omitempty"`
}

func WriteArchive(w io.Writer, root, name string, origin ArchiveOrigin) error {
	if !validSkillName.MatchString(name) {
		return errors.New("invalid skill name")
	}
	archive := zip.NewWriter(w)
	if origin != (ArchiveOrigin{}) {
		metadata, err := archive.Create(".skillhub.json")
		if err != nil {
			return err
		}
		if err := json.NewEncoder(metadata).Encode(origin); err != nil {
			return err
		}
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer directory.Close()
	err = fs.WalkDir(directory.FS(), ".", func(relative string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !entry.IsDir() && !info.Mode().IsRegular() {
			return errors.New("invalid file in snapshot")
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = path.Join(name, relative)
		if entry.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		target, err := archive.CreateHeader(header)
		if err != nil || entry.IsDir() {
			return err
		}
		file, err := directory.Open(filepath.FromSlash(relative))
		if err != nil {
			return err
		}
		defer file.Close()
		written, err := io.Copy(target, io.LimitReader(file, info.Size()+1))
		if err == nil && written != info.Size() {
			return errors.New("snapshot changed during export")
		}
		return err
	})
	if err != nil {
		return err
	}
	return archive.Close()
}

func ExtractArchive(file, destination string) error {
	archive, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	defer archive.Close()
	if len(archive.File) > 20000 {
		return errors.New("ZIP 文件数超过上限")
	}
	seen := make(map[string]bool, len(archive.File))
	remaining := int64(128 << 20)
	for _, entry := range archive.File {
		name := strings.TrimSuffix(entry.Name, "/")
		key := strings.ToLower(name)
		if name == "." || !fs.ValidPath(name) || strings.ContainsAny(name, "\\\x00:") || seen[key] {
			return fmt.Errorf("无效或重复的 ZIP 路径：%s", name)
		}
		seen[key] = true
		target := filepath.Join(destination, filepath.FromSlash(name))
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if !entry.Mode().IsRegular() {
			return errors.New("ZIP 只支持普通文件和目录")
		}
		limit := min(remaining, 20<<20)
		if entry.UncompressedSize64 > uint64(limit) {
			return errors.New("ZIP 解包大小超过上限")
		}
		n, err := extractFile(entry, target, limit)
		if err != nil {
			return err
		}
		remaining -= n
	}
	return nil
}

func extractFile(entry *zip.File, target string, limit int64) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return 0, err
	}
	source, err := entry.Open()
	if err != nil {
		return 0, err
	}
	defer source.Close()
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600|entry.Mode().Perm()&0100)
	if err != nil {
		return 0, err
	}
	size, copyErr := io.Copy(file, io.LimitReader(source, limit+1))
	if size > limit {
		copyErr = errors.New("ZIP 解包大小超过上限")
	}
	return size, errors.Join(copyErr, file.Close())
}

func ArchiveCandidates(root string) ([]ScannedSkill, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var candidates []ScannedSkill
	err = fs.WalkDir(os.DirFS(root), ".", func(relative string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if relative != "." && strings.HasPrefix(entry.Name(), ".") {
			return fs.SkipDir
		}
		skill, ok, err := readSkill(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || !ok {
			return err
		}
		for _, parent := range candidates {
			if parent.Path == "." || strings.HasPrefix(relative, parent.Path+"/") {
				return errors.New("ZIP 包含嵌套的 Skill 根目录")
			}
		}
		if len(candidates) >= 100 {
			return errors.New("ZIP 最多收录 100 个 Skill")
		}
		skill.Path = relative
		candidates = append(candidates, skill)
		return nil
	})
	return candidates, err
}
