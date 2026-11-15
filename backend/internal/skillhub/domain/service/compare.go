package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

type FileDifference struct {
	Path, Kind, LeftText, RightText string
	Binary                          bool
}

// StoredPath resolves a managed snapshot without allowing paths outside the data root.
func (i *Intake) StoredPath(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("invalid stored path")
	}
	root, err := filepath.EvalSymlinks(i.root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, path))
	if err != nil || !within(root, resolved) {
		return "", fmt.Errorf("stored path unavailable")
	}
	return resolved, nil
}

// CompareDirectories compares complete Skill directories without changing either source.
func CompareDirectories(left, right string) ([]FileDifference, string, string, error) {
	leftFiles, leftDigest, err := PreviewLocalSkill(left)
	if err != nil {
		return nil, "", "", err
	}
	rightFiles, rightDigest, err := PreviewLocalSkill(right)
	if err != nil {
		return nil, "", "", err
	}
	leftMap, rightMap := make(map[string]ImportFile), make(map[string]ImportFile)
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
		if hasOld && hasNew && old.Digest == newer.Digest {
			continue
		}
		total += old.Size + newer.Size
		if total > 8*1024*1024 {
			return nil, "", "", fmt.Errorf("comparison exceeds 8 MiB")
		}
		if old.Size > 1024*1024 || newer.Size > 1024*1024 {
			return nil, "", "", fmt.Errorf("file exceeds 1 MiB: %s", path)
		}
		change := FileDifference{Path: path, Kind: "modified"}
		if !hasOld {
			change.Kind = "added"
		}
		if !hasNew {
			change.Kind = "deleted"
		}
		var before, after []byte
		if hasOld {
			before, err = os.ReadFile(filepath.Join(left, filepath.FromSlash(path)))
			if err != nil {
				return nil, "", "", err
			}
		}
		if hasNew {
			after, err = os.ReadFile(filepath.Join(right, filepath.FromSlash(path)))
			if err != nil {
				return nil, "", "", err
			}
		}
		if hasOld {
			hash := sha256.Sum256(before)
			if hex.EncodeToString(hash[:]) != old.Digest {
				return nil, "", "", fmt.Errorf("source changed during comparison")
			}
		}
		if hasNew {
			hash := sha256.Sum256(after)
			if hex.EncodeToString(hash[:]) != newer.Digest {
				return nil, "", "", fmt.Errorf("source changed during comparison")
			}
		}
		change.Binary = !utf8.Valid(before) || !utf8.Valid(after) || bytes.ContainsRune(before, 0) || bytes.ContainsRune(after, 0)
		if !change.Binary {
			change.LeftText, change.RightText = string(before), string(after)
		}
		changes = append(changes, change)
	}
	return changes, leftDigest, rightDigest, nil
}
