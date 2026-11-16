package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

type FileDifference struct {
	Path, Kind, LeftText, RightText, Detail string
	Binary                                  bool
}

// CompareDirectories compares complete Skill directories without changing either source.
func CompareDirectories(left, right string) ([]FileDifference, string, string, error) {
	leftFiles, leftDigest, err := readSkillFiles(left)
	if err != nil {
		return nil, "", "", err
	}
	rightFiles, rightDigest, err := readSkillFiles(right)
	if err != nil {
		return nil, "", "", err
	}
	beforeRoot, err := os.OpenRoot(left)
	if err != nil {
		return nil, "", "", err
	}
	defer beforeRoot.Close()
	afterRoot, err := os.OpenRoot(right)
	if err != nil {
		return nil, "", "", err
	}
	defer afterRoot.Close()

	leftMap, rightMap := make(map[string]skillFile), make(map[string]skillFile)
	paths := make(map[string]bool)
	for _, file := range leftFiles {
		leftMap[file.Path] = file
		paths[file.Path] = true
	}
	for _, file := range rightFiles {
		rightMap[file.Path] = file
		paths[file.Path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)

	changes := make([]FileDifference, 0)
	var total int64
	for _, path := range ordered {
		old, hasOld := leftMap[path]
		newer, hasNew := rightMap[path]
		if hasOld && hasNew && old.Digest == newer.Digest && old.Mode == newer.Mode {
			continue
		}
		change := FileDifference{Path: path, Kind: "modified"}
		switch {
		case !hasOld:
			change.Kind = "added"
		case !hasNew:
			change.Kind = "deleted"
		case old.Mode != newer.Mode:
			change.Detail = "执行权限已变更"
		}
		if hasOld && hasNew && old.Digest == newer.Digest {
			changes = append(changes, change)
			continue
		}
		size := old.Size + newer.Size
		if old.Size > 1<<20 || newer.Size > 1<<20 || total+size > 8<<20 {
			change.Detail = "文件变更超出行差异预览上限"
			changes = append(changes, change)
			continue
		}
		total += size
		before, err := readComparedFile(beforeRoot, old)
		if err != nil {
			return nil, "", "", err
		}
		after, err := readComparedFile(afterRoot, newer)
		if err != nil {
			return nil, "", "", err
		}
		change.Binary = !utf8.Valid(before) || !utf8.Valid(after) || bytes.ContainsRune(before, 0) || bytes.ContainsRune(after, 0)
		if !change.Binary {
			change.LeftText, change.RightText = string(before), string(after)
		}
		changes = append(changes, change)
	}
	return changes, leftDigest, rightDigest, nil
}

func readComparedFile(root *os.Root, file skillFile) ([]byte, error) {
	if file.Path == "" {
		return nil, nil
	}
	handle, err := root.Open(filepath.FromSlash(file.Path))
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	data, err := io.ReadAll(io.LimitReader(handle, file.Size+1))
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != file.Digest {
		return nil, fmt.Errorf("source changed during comparison: %s", file.Path)
	}
	return data, nil
}
