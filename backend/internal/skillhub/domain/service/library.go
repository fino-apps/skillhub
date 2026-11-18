package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

var (
	ErrInvalid  = errors.New("invalid library input")
	ErrConflict = errors.New("library state changed or conflicts with the operation")
)

type Library struct {
	store                       repo.Library
	snapshots, uploads, exports string
	writes                      sync.Mutex
	commits                     sync.Mutex
}

type LibraryDeps struct {
	Store       repo.Library
	SnapshotDir string
	UploadDir   string
	ExportDir   string
}

func NewLibrary(deps LibraryDeps) *Library {
	return &Library{
		store:     deps.Store,
		snapshots: deps.SnapshotDir,
		uploads:   deps.UploadDir,
		exports:   deps.ExportDir,
	}
}

type SkillDetails struct {
	Skill   model.SkillSnapshot
	Tags    []string
	Default *model.SkillVersionSnapshot
}

func (s *Library) Skills(ctx context.Context, filter repo.SkillFilter, size int, token string) ([]SkillDetails, string, error) {
	rows, next, err := s.store.Skills(ctx, filter, size, token)
	if err != nil {
		return nil, "", err
	}
	details, err := s.details(ctx, rows)
	return details, next, err
}

func (s *Library) Skill(ctx context.Context, id string) (model.SkillSnapshot, error) {
	return s.store.Skill(ctx, id)
}

func (s *Library) Detail(ctx context.Context, id string) (SkillDetails, error) {
	row, err := s.store.Skill(ctx, id)
	if err != nil {
		return SkillDetails{}, err
	}
	results, err := s.details(ctx, []model.SkillSnapshot{row})
	if err != nil {
		return SkillDetails{}, err
	}
	return results[0], nil
}

func (s *Library) details(ctx context.Context, rows []model.SkillSnapshot) ([]SkillDetails, error) {
	result := make([]SkillDetails, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}

	ids := make([]string, 0, len(rows))
	var versions []string
	for _, row := range rows {
		ids = append(ids, string(row.ID))
		if row.DefaultVersionId != "" {
			versions = append(versions, row.DefaultVersionId)
		}
	}

	tags, err := s.store.Tags(ctx, ids)
	if err != nil {
		return nil, err
	}

	defaults, err := s.store.DefaultVersions(ctx, versions)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*model.SkillVersionSnapshot, len(defaults))
	for i := range defaults {
		byID[string(defaults[i].ID)] = &defaults[i]
	}

	for _, row := range rows {
		version := byID[row.DefaultVersionId]
		if version != nil && version.SkillId != string(row.ID) {
			version = nil
		}
		result = append(result, SkillDetails{row, tags[string(row.ID)], version})
	}
	return result, nil
}

func (s *Library) Facets(ctx context.Context) (repo.Facets, error) {
	return s.store.Facets(ctx)
}

type Metadata struct {
	Category, Purpose, Notes, UpstreamName, UpstreamURL string
	Tags                                                []string
}

type MetadataChange struct {
	SkillID, Revision   string
	Category, Purpose   *string
	AddTags, RemoveTags []string
}

func normalizeTags(values []string) ([]string, error) {
	if len(values) > 100 {
		return nil, ErrInvalid
	}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.TrimSpace(value)
		if tag == "" || utf8.RuneCountInString(tag) > 128 {
			return nil, ErrInvalid
		}
		cleaned = append(cleaned, tag)
	}
	slices.Sort(cleaned)
	return slices.Compact(cleaned), nil
}

func (s *Library) ReplaceMetadata(ctx context.Context, id string, input Metadata) (SkillDetails, error) {
	tags, err := normalizeTags(input.Tags)
	if err != nil {
		return SkillDetails{}, err
	}
	err = s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		skill, err := tx.Skill(id)
		if err != nil {
			return err
		}
		skill.Category, skill.Purpose = strings.TrimSpace(input.Category), strings.TrimSpace(input.Purpose)
		skill.Notes, skill.UpstreamName, skill.UpstreamUrl = input.Notes, strings.TrimSpace(input.UpstreamName), strings.TrimSpace(input.UpstreamURL)
		skill.UpdateTime = core.Now()
		if err := tx.UpdateSkill(skill, "category", "purpose", "notes", "upstreamName", "upstreamUrl", "updateTime"); err != nil {
			return err
		}
		return tx.ChangeTags(id, true, tags, nil)
	})
	if err != nil {
		return SkillDetails{}, err
	}
	return s.Detail(ctx, id)
}

func (s *Library) Organize(ctx context.Context, changes []MetadataChange) error {
	return s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		for _, change := range changes {
			if _, err := uuid.Parse(change.SkillID); err != nil {
				return ErrInvalid
			}
			skill, err := tx.Skill(change.SkillID)
			if err != nil {
				return err
			}
			if change.Revision != "" && change.Revision != skill.UpdateTime.Format() {
				return ErrConflict
			}

			fields := []string{"updateTime"}
			if change.Category != nil {
				skill.Category = strings.TrimSpace(*change.Category)
				if utf8.RuneCountInString(skill.Category) > 128 {
					return ErrInvalid
				}
				fields = append(fields, "category")
			}
			if change.Purpose != nil {
				skill.Purpose = strings.TrimSpace(*change.Purpose)
				if utf8.RuneCountInString(skill.Purpose) > 240 {
					return ErrInvalid
				}
				fields = append(fields, "purpose")
			}

			add, err := normalizeTags(change.AddTags)
			if err != nil {
				return err
			}
			remove, err := normalizeTags(change.RemoveTags)
			if err != nil {
				return err
			}
			skill.UpdateTime = core.Now()
			if err := tx.UpdateSkill(skill, fields...); err != nil {
				return err
			}

			if err := tx.ChangeTags(change.SkillID, false, add, remove); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Library) Versions(ctx context.Context, skill string, size int, token string) ([]model.SkillVersionSnapshot, string, error) {
	if _, err := s.store.Skill(ctx, skill); err != nil {
		return nil, "", err
	}
	return s.store.Versions(ctx, skill, size, token)
}

func (s *Library) Version(ctx context.Context, skill, id string) (model.SkillVersionSnapshot, error) {
	return s.store.Version(ctx, skill, id)
}

func setDefault(tx repo.LibraryTx, skill model.SkillSnapshot, version model.SkillVersionSnapshot) error {
	skill.DefaultVersionId, skill.Description, skill.UpdateTime = string(version.ID), version.Description, core.Now()
	return tx.UpdateSkill(skill, "defaultVersionId", "description", "updateTime")
}

func (s *Library) SetDefault(ctx context.Context, skillID, versionID string) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	return s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		skill, err := tx.Skill(skillID)
		if err != nil {
			return err
		}
		version, err := tx.Version(skillID, versionID)
		if err != nil {
			return err
		}
		return setDefault(tx, skill, version)
	})
}

func (s *Library) DeleteVersion(ctx context.Context, skillID, versionID string) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	err := s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		skill, err := tx.Skill(skillID)
		if err != nil {
			return err
		}
		if skill.DefaultVersionId == versionID {
			return ErrConflict
		}
		version, err := tx.Version(skillID, versionID)
		if err != nil {
			return err
		}
		return tx.DeleteVersion(version)
	})
	if err == nil {
		_ = os.Remove(s.snapshotPath(versionID))
	}
	return err
}

func (s *Library) DeleteSkill(ctx context.Context, id string) error {
	return s.deleteSkills(ctx, []string{id})
}

func (s *Library) BatchDeleteSkills(ctx context.Context, ids []string) error {
	if len(ids) == 0 || len(ids) > 100 {
		return ErrInvalid
	}
	unique := make(map[uuid.UUID]struct{}, len(ids))
	skillIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil || len(id) != 36 {
			return ErrInvalid
		}
		if _, exists := unique[parsed]; exists {
			continue
		}
		unique[parsed] = struct{}{}
		skillIDs = append(skillIDs, parsed.String())
	}
	return s.deleteSkills(ctx, skillIDs)
}

func (s *Library) deleteSkills(ctx context.Context, ids []string) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	var versions []string
	err := s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		for _, id := range ids {
			skill, err := tx.Skill(id)
			if err != nil {
				return err
			}
			removed, err := tx.DeleteSkill(skill)
			if err != nil {
				return err
			}
			versions = append(versions, removed...)
		}
		return nil
	})
	if err == nil {
		for _, id := range versions {
			_ = os.Remove(s.snapshotPath(id))
		}
	}
	return err
}

func (s *Library) snapshotPath(id string) string {
	return filepath.Join(s.snapshots, id+".zip")
}

func (s *Library) Available(version model.SkillVersionSnapshot) bool {
	info, err := os.Stat(s.snapshotPath(string(version.ID)))
	return err == nil && info.Mode().IsRegular()
}

func (s *Library) ReadArchive(version model.SkillVersionSnapshot) ([]byte, error) {
	if _, err := uuid.Parse(string(version.ID)); err != nil {
		return nil, repo.ErrNotFound
	}
	data, err := os.ReadFile(s.snapshotPath(string(version.ID)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, repo.ErrNotFound
	}
	return data, err
}
