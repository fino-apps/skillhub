package service

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

func (s *Library) snapshotChecks() func(model.SkillVersionSnapshot) bool {
	checked := make(map[model.SkillVersionID]bool)
	return func(version model.SkillVersionSnapshot) bool {
		if valid, ok := checked[version.ID]; ok {
			return valid
		}
		valid := s.usableSnapshot(version)
		checked[version.ID] = valid
		return valid
	}
}

func (s *Library) usableSnapshot(version model.SkillVersionSnapshot) bool {
	id := string(version.ID)
	if !canonicalBackupID(id) {
		return false
	}
	path := s.snapshotPath(id)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	file, err := os.Open(path) // #nosec G304 -- path is built from a validated snapshot UUID.
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	return validateSnapshotArchive(file, info.Size(), version.ContentDigest) == nil
}

// snapshotChanges retains replaced files until the database transaction commits.
type snapshotChanges struct {
	library   *Library
	originals map[string]string
}

func (c *snapshotChanges) write(id model.SkillVersionID, data []byte) error {
	if !canonicalBackupID(string(id)) {
		return ErrInvalid
	}
	path := c.library.snapshotPath(string(id))
	if c.originals == nil {
		c.originals = make(map[string]string)
	}
	if _, staged := c.originals[path]; !staged {
		var original string
		info, err := os.Lstat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return err
		case !info.Mode().IsRegular():
			return fmt.Errorf("invalid snapshot file: %s", id)
		default:
			var err error
			original, err = c.retain(path)
			if err != nil {
				return err
			}
		}
		c.originals[path] = original
	}
	return c.library.writeSnapshot(string(id), data)
}

func (c *snapshotChanges) retain(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304 -- path is built from a validated snapshot UUID.
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	holding, err := os.CreateTemp(c.library.snapshots, ".before-merge-")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(holding, file)
	if err = errors.Join(err, holding.Close()); err != nil {
		_ = os.Remove(holding.Name())
		return "", err
	}
	return holding.Name(), nil
}

func (c *snapshotChanges) finish(transactionErr error) error {
	for path, original := range c.originals {
		switch {
		case transactionErr == nil:
			if original != "" {
				_ = os.Remove(original)
			}
		case original != "":
			transactionErr = errors.Join(transactionErr, os.Rename(original, path))
		default:
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				transactionErr = errors.Join(transactionErr, err)
			}
		}
	}
	return transactionErr
}
