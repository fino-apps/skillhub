package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

type Source struct {
	Kind, Repository, Path, Ref, Commit string
}

type VersionInput struct {
	Notes, ExpectedDigest string
	MakeDefault           bool
}

func (s *Library) saveDirectory(ctx context.Context, dir, target string, source Source, input VersionInput) (model.SkillVersionSnapshot, bool, error) {
	manifest, err := ReadManifest(dir)
	if err != nil {
		return model.SkillVersionSnapshot{}, false, invalid(err)
	}

	data, digest, err := Archive(dir)
	if err != nil {
		return model.SkillVersionSnapshot{}, false, invalid(err)
	}
	if input.ExpectedDigest != "" && input.ExpectedDigest != digest {
		return model.SkillVersionSnapshot{}, false, ErrConflict
	}

	value := dir
	switch source.Kind {
	case "github":
		value = source.Repository + "\x00" + source.Path
	case "zip":
		value = digest
	}

	key := fmt.Sprintf("%x", sha256.Sum256([]byte(source.Kind+"\x00"+value)))
	s.writes.Lock()
	defer s.writes.Unlock()
	now := core.Now()
	var version model.SkillVersionSnapshot
	var savedFile string
	var created bool
	err = s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		var skill model.SkillSnapshot
		if target != "" {
			row, err := tx.Skill(target)
			if err != nil {
				return err
			}
			skill = row
		} else {
			if _, err := tx.SkillBySource(key); err == nil {
				return nil
			} else if !errors.Is(err, repo.ErrNotFound) {
				return err
			}

			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			skill = model.SkillSnapshot{ID: model.SkillID(id.String()), Name: manifest.Name, Description: manifest.Description,
				SourceKey: key, ImportKind: source.Kind, RepositoryUrl: source.Repository, RepositoryPath: source.Path,
				CreateTime: now, UpdateTime: now}
			if source.Kind == "folder" {
				skill.LocalPath = dir
			}
			if source.Kind == "github" {
				skill.UpstreamUrl = source.Repository
			}
			if err := tx.CreateSkill(skill); err != nil {
				return err
			}
		}

		existing, err := tx.MatchingVersion(string(skill.ID), source.Commit, digest)
		switch {
		case err == nil:
			version = existing
		case errors.Is(err, repo.ErrNotFound):
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			version = model.SkillVersionSnapshot{ID: model.SkillVersionID(id.String()), SkillId: string(skill.ID),
				Name: manifest.Name, Description: manifest.Description, Notes: input.Notes, ContentDigest: digest,
				SourceRef: source.Ref, CommitSha: source.Commit, CreateTime: now, UpdateTime: now}
			savedFile = s.snapshotPath(string(version.ID))
			if err := os.WriteFile(savedFile, data, 0600); err != nil {
				return err
			}

			if err := tx.CreateVersion(version); err != nil {
				return err
			}
			created = true
		default:
			return err
		}

		if skill.DefaultVersionId == "" || input.MakeDefault {
			return setDefault(tx, skill, version)
		}
		return nil
	})
	if err != nil && savedFile != "" {
		_ = os.Remove(savedFile)
	}
	return version, created, err
}

func (s *Library) Repair(ctx context.Context, skillID, versionID string) error {
	skill, err := s.store.Skill(ctx, skillID)
	if err != nil {
		return err
	}

	row, err := s.store.Version(ctx, skillID, versionID)
	if err != nil {
		return err
	}
	if s.Available(row) {
		return nil
	}
	if skill.RepositoryUrl == "" || !CommitPattern.MatchString(row.CommitSha) {
		return ErrConflict
	}

	root, _, cleanup, err := Checkout(ctx, skill.RepositoryUrl, row.CommitSha)
	if err != nil {
		return invalid(err)
	}
	defer cleanup()
	dir, err := Within(root, skill.RepositoryPath)
	if err != nil {
		return invalid(err)
	}

	data, digest, err := Archive(dir)
	if err != nil {
		return invalid(err)
	}
	if row.ContentDigest != "" && row.ContentDigest != digest {
		return ErrConflict
	}
	s.writes.Lock()
	defer s.writes.Unlock()
	if _, err := s.store.Version(ctx, skillID, versionID); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(s.snapshots, ".repair-")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	_, err = temporary.Write(data)
	if err = errors.Join(err, temporary.Close()); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), s.snapshotPath(versionID))
}

func invalid(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalid, err)
}
