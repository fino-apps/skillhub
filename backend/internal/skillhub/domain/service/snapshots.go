package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var ErrSnapshotChanged = errors.New("skill changed after comparison")

type SnapshotStore struct{ root string }

func NewSnapshotStore(path string) (*SnapshotStore, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	return &SnapshotStore{root: root}, nil
}

type SkillSnapshot struct {
	ScannedSkill
	Digest    string
	path      string
	published bool
}

func (s *SnapshotStore) Capture(path, expectedDigest string) (*SkillSnapshot, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("skill path must be absolute")
	}
	original, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if within(s.root, original) || within(original, s.root) {
		return nil, fmt.Errorf("skill directory overlaps with snapshot storage")
	}
	staging, err := os.MkdirTemp(s.root, ".snapshot-")
	if err != nil {
		return nil, err
	}
	snapshot := &SkillSnapshot{path: staging}
	defer func() {
		if snapshot.Digest == "" {
			snapshot.Discard()
		}
	}()
	_, digest, err := inspectSkillFiles(original, staging)
	if err != nil {
		return nil, err
	}
	if expectedDigest != "" && digest != expectedDigest {
		return nil, ErrSnapshotChanged
	}
	skill, ok, err := readSkill(staging)
	if err != nil || !ok || len(skill.Name) > 64 || !validSkillName.MatchString(skill.Name) || len(skill.Description) > 1024 {
		return nil, fmt.Errorf("invalid SKILL.md")
	}
	skill.Path = original
	snapshot.ScannedSkill, snapshot.Digest = skill, digest
	return snapshot, nil
}

// Publish places a complete version at a path independent of the original project.
func (s *SnapshotStore) Publish(snapshot *SkillSnapshot, skillID, versionID string) error {
	target, err := s.versionPath(skillID, versionID, snapshot.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	if err := os.Rename(snapshot.path, target); err != nil {
		return err
	}
	snapshot.path = target
	snapshot.published = true
	return nil
}

func (s *SkillSnapshot) Discard() {
	if s.published {
		version := filepath.Dir(s.path)
		_ = os.RemoveAll(version)
		_ = os.Remove(filepath.Dir(version))
		return
	}
	_ = os.RemoveAll(s.path)
}

func (s *SnapshotStore) Path(skillID, versionID, name string) (string, error) {
	target, err := s.versionPath(skillID, versionID, name)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil || resolved != target {
		return "", fmt.Errorf("version snapshot unavailable")
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		return "", fmt.Errorf("version snapshot unavailable")
	}
	return target, nil
}

func (s *SnapshotStore) RemoveSkill(skillID string) error {
	if uuid.Validate(skillID) != nil {
		return fmt.Errorf("invalid skill ID")
	}
	return os.RemoveAll(filepath.Join(s.root, skillID))
}

func (s *SnapshotStore) versionPath(skillID, versionID, name string) (string, error) {
	if uuid.Validate(skillID) != nil || uuid.Validate(versionID) != nil || len(name) > 64 || !validSkillName.MatchString(name) {
		return "", fmt.Errorf("invalid snapshot identity")
	}
	return filepath.Join(s.root, skillID, versionID, name), nil
}
