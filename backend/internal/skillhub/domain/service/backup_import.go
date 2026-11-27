package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/safecount"
	"github.com/google/uuid"
)

const stagedBackupName = "backup.zip"

type MergeSkillEntry struct {
	ID, Name, Source, DefaultVersion string
	VersionCount                     int32
}

type BackupMergeCandidate struct {
	TargetID string
	Summary  VersionMergeSummary
}

type BackupImportItem struct {
	Skill         MergeSkillEntry
	Candidates    []BackupMergeCandidate
	ExactTargetID string
}

type BackupPreview struct {
	UploadID string
	Items    []BackupImportItem
	Targets  []MergeSkillEntry
}

type BackupChoice struct {
	SkillID, Action, TargetID string
}

type BackupCommitResult struct {
	Added, Merged, Skipped int32
	Versions               VersionMergeSummary
}

func versionsBySkill(versions []model.SkillVersionSnapshot) map[string][]model.SkillVersionSnapshot {
	result := make(map[string][]model.SkillVersionSnapshot)
	for _, version := range versions {
		result[version.SkillId] = append(result[version.SkillId], version)
	}
	return result
}

func mergeEntry(skill model.SkillSnapshot, versions []model.SkillVersionSnapshot) MergeSkillEntry {
	entry := MergeSkillEntry{ID: string(skill.ID), Name: skill.Name, Source: skill.ImportKind, VersionCount: safecount.Int32(int64(len(versions)))}
	switch {
	case skill.RepositoryUrl != "":
		entry.Source = skill.RepositoryUrl
		if skill.RepositoryPath != "" {
			entry.Source += " · " + skill.RepositoryPath
		}
	case skill.LocalPath != "":
		entry.Source = skill.LocalPath
	case skill.UpstreamUrl != "":
		entry.Source = skill.UpstreamUrl
	}
	for _, version := range versions {
		if skill.DefaultVersionId == string(version.ID) {
			entry.DefaultVersion = version.CommitSha
			if entry.DefaultVersion == "" {
				entry.DefaultVersion = version.ContentDigest
			}
			if entry.DefaultVersion == "" {
				entry.DefaultVersion = string(version.ID)
			}
			break
		}
	}
	return entry
}

func matchSkillKeys(skill model.SkillSnapshot, versions []model.SkillVersionSnapshot, aliases []string) []string {
	keys := []string{"id:" + string(skill.ID)}
	if skill.SourceKey != "" {
		keys = append(keys, "source:"+skill.SourceKey)
	}
	for _, alias := range aliases {
		keys = append(keys, "source:"+alias)
	}
	if name := strings.ToLower(strings.TrimSpace(skill.Name)); name != "" {
		keys = append(keys, "name:"+name)
	}
	if skill.RepositoryUrl != "" {
		keys = append(keys, "origin:"+strings.ToLower(strings.TrimSuffix(skill.RepositoryUrl, "/"))+"\x00"+skill.RepositoryPath)
	} else if skill.UpstreamUrl != "" {
		keys = append(keys, "origin:"+strings.ToLower(strings.TrimSuffix(skill.UpstreamUrl, "/"))+"\x00")
	}
	for _, version := range versions {
		if version.ContentDigest != "" {
			keys = append(keys, "digest:"+version.ContentDigest)
		}
	}
	return keys
}

func (s *Library) PreviewBackupImport(ctx context.Context, data []byte) (BackupPreview, error) {
	backup, err := parseBackup(data)
	if err != nil {
		return BackupPreview{}, err
	}
	s.writes.Lock()
	result, err := s.previewBackup(ctx, backup)
	s.writes.Unlock()
	if err != nil {
		return BackupPreview{}, err
	}

	s.commits.Lock()
	defer s.commits.Unlock()
	if err := s.cleanUploads(time.Now()); err != nil {
		return BackupPreview{}, err
	}
	result.UploadID = uuid.NewString()
	dir := filepath.Join(s.uploads, result.UploadID)
	if err := os.Mkdir(dir, 0700); err != nil {
		return BackupPreview{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, stagedBackupName), data, 0600); err != nil {
		_ = os.RemoveAll(dir)
		return BackupPreview{}, err
	}
	return result, nil
}

func (s *Library) previewBackup(ctx context.Context, backup backupArchive) (BackupPreview, error) {
	var current repo.LibraryData
	if err := s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		var err error
		current, err = tx.BackupData()
		return err
	}); err != nil {
		return BackupPreview{}, err
	}
	var result BackupPreview
	currentVersions, incomingVersions := versionsBySkill(current.Versions), versionsBySkill(backup.Versions)
	currentAliases, incomingAliases := sourceAliasesBySkill(current.Aliases), sourceAliasesBySkill(backup.Aliases)
	defaults := make(map[string]string, len(current.Skills))
	currentUsable := s.snapshotChecks()
	index := make(map[string]map[string]bool)
	for _, skill := range current.Skills {
		id := string(skill.ID)
		versions := currentVersions[id]
		result.Targets = append(result.Targets, mergeEntry(skill, versions))
		defaults[id] = skill.DefaultVersionId
		for _, key := range matchSkillKeys(skill, versions, currentAliases[id]) {
			if index[key] == nil {
				index[key] = make(map[string]bool)
			}
			index[key][id] = true
		}
	}
	for _, skill := range backup.Skills {
		versions := incomingVersions[string(skill.ID)]
		item := BackupImportItem{Skill: mergeEntry(skill, versions)}
		all, exact := make(map[string]bool), make(map[string]bool)
		for _, key := range matchSkillKeys(skill, versions, incomingAliases[string(skill.ID)]) {
			for id := range index[key] {
				all[id] = true
				if strings.HasPrefix(key, "id:") || strings.HasPrefix(key, "source:") {
					exact[id] = true
				}
			}
		}
		if len(exact) == 1 {
			for id := range exact {
				item.ExactTargetID = id
			}
		}
		candidates := make([]string, 0, len(all))
		for id := range all {
			candidates = append(candidates, id)
		}
		sort.Strings(candidates)
		for _, id := range candidates {
			plan := planVersionMerge(versions, currentVersions[id], defaults[id], backup.hasSnapshot, currentUsable)
			item.Candidates = append(item.Candidates, BackupMergeCandidate{TargetID: id, Summary: plan.summary})
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// openStagedBackup keeps the file open because ZIP entries read from it on demand.
func (s *Library) openStagedBackup(uploadID string) (backupArchive, *os.File, error) {
	if !canonicalBackupID(uploadID) {
		return backupArchive{}, nil, ErrInvalid
	}
	file, err := os.Open(filepath.Join(s.uploads, uploadID, stagedBackupName)) // #nosec G304 -- uploadID is a canonical UUID.
	if errors.Is(err, os.ErrNotExist) {
		return backupArchive{}, nil, repo.ErrNotFound
	}
	if err != nil {
		return backupArchive{}, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return backupArchive{}, nil, err
	}
	if time.Since(info.ModTime()) >= uploadLifetime {
		_ = file.Close()
		return backupArchive{}, nil, repo.ErrNotFound
	}
	backup, err := parseBackupArchive(file, info.Size())
	if err != nil {
		_ = file.Close()
		return backupArchive{}, nil, err
	}
	return backup, file, nil
}

func (s *Library) PreviewBackupMerge(ctx context.Context, uploadID, sourceID, targetID string) (VersionMergeSummary, error) {
	var summary VersionMergeSummary
	if !canonicalBackupID(uploadID) {
		return summary, ErrInvalid
	}
	s.commits.Lock()
	defer s.commits.Unlock()

	backup, file, err := s.openStagedBackup(uploadID)
	if err != nil {
		return summary, err
	}
	defer func() { _ = file.Close() }()
	sourceIndex := slices.IndexFunc(backup.Skills, func(skill model.SkillSnapshot) bool { return string(skill.ID) == sourceID })
	if sourceIndex < 0 {
		return summary, repo.ErrNotFound
	}
	source := backup.Skills[sourceIndex]
	var incoming []model.SkillVersionSnapshot
	for _, version := range backup.Versions {
		if version.SkillId == sourceID {
			incoming = append(incoming, version)
		}
	}
	s.writes.Lock()
	defer s.writes.Unlock()
	err = s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		target, err := tx.Skill(targetID)
		if err != nil {
			return err
		}
		current, err := tx.SkillVersions(targetID)
		if err != nil {
			return err
		}
		plan := planVersionMerge(incoming, current, target.DefaultVersionId, backup.hasSnapshot, s.snapshotChecks())
		summary = plan.details(source)
		return nil
	})
	return summary, err
}

func (s *Library) CommitBackupImport(ctx context.Context, uploadID string, choices []BackupChoice) (BackupCommitResult, error) {
	var result BackupCommitResult
	if !canonicalBackupID(uploadID) {
		return result, ErrInvalid
	}
	s.commits.Lock()
	defer s.commits.Unlock()
	if err := s.cleanUploads(time.Now()); err != nil {
		return result, err
	}
	backup, file, err := s.openStagedBackup(uploadID)
	if err != nil {
		return result, err
	}

	result, err = s.applyBackup(ctx, backup, choices, false)
	_ = file.Close()
	if err == nil {
		_ = os.RemoveAll(filepath.Join(s.uploads, uploadID))
	}
	return result, err
}

func (s *Library) applyBackup(ctx context.Context, backup backupArchive, choices []BackupChoice, skipExisting bool) (BackupCommitResult, error) {
	var result BackupCommitResult
	decisions, err := backupDecisions(backup.backupManifest, choices)
	if err != nil {
		return result, err
	}

	s.writes.Lock()
	defer s.writes.Unlock()
	bySkill := versionsBySkill(backup.Versions)
	aliases := sourceAliasesBySkill(backup.Aliases)
	tags := make(map[string][]model.SkillTagSnapshot)
	for _, tag := range backup.Tags {
		tags[tag.SkillId] = append(tags[tag.SkillId], tag)
	}
	changes := snapshotChanges{library: s}
	restore := func(version model.SkillVersionSnapshot, target model.SkillVersionID) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		file := backup.snapshot(version)
		if file == nil {
			return nil
		}
		data, err := readBackupSnapshot(file, version)
		if err != nil {
			return err
		}
		return changes.write(target, data)
	}
	err = s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		for _, skill := range backup.Skills {
			id := string(skill.ID)
			decision := decisions[id]
			versions := bySkill[id]
			switch decision.Action {
			case "skip":
				result.Skipped++
			case "merge":
				target, err := tx.Skill(decision.TargetID)
				if err != nil {
					return err
				}
				current, err := tx.SkillVersions(decision.TargetID)
				if err != nil {
					return err
				}
				plan := planVersionMerge(versions, current, target.DefaultVersionId, backup.hasSnapshot, s.snapshotChecks())
				if err := applyVersionMerge(plan, skill, target, tx,
					func(version model.SkillVersionSnapshot) (model.SkillVersionSnapshot, error) {
						created, err := createImportedVersion(tx, version)
						if err == nil {
							err = restore(version, created.ID)
						}
						return created, err
					}, restore); err != nil {
					return err
				}
				for _, key := range append(aliases[id], skill.SourceKey) {
					if err := rememberSource(tx, decision.TargetID, key, true); err != nil {
						return err
					}
				}
				result.Merged++
				result.Versions.add(plan.summary)
			default:
				err := importIndependentSkill(tx, skill, versions, tags[id], aliases[id], decision.Action == "add", restore)
				switch {
				case err == nil:
					result.Added++
				case skipExisting && errors.Is(err, ErrConflict):
					result.Skipped++
				default:
					return err
				}
			}
		}
		return nil
	})
	if err = changes.finish(err); err != nil {
		return BackupCommitResult{}, err
	}
	return result, nil
}

func backupDecisions(manifest backupManifest, choices []BackupChoice) (map[string]BackupChoice, error) {
	skills := make(map[string]bool, len(manifest.Skills))
	for _, skill := range manifest.Skills {
		skills[string(skill.ID)] = true
	}
	decisions := make(map[string]BackupChoice, len(choices))
	for _, choice := range choices {
		if !skills[choice.SkillID] || decisions[choice.SkillID].SkillID != "" {
			return nil, ErrInvalid
		}
		switch choice.Action {
		case "add", "skip":
			if choice.TargetID != "" {
				return nil, ErrInvalid
			}
		case "merge":
			if !canonicalBackupID(choice.TargetID) {
				return nil, ErrInvalid
			}
		default:
			return nil, ErrInvalid
		}
		decisions[choice.SkillID] = choice
	}
	return decisions, nil
}

func importIndependentSkill(tx repo.LibraryTx, skill model.SkillSnapshot, versions []model.SkillVersionSnapshot,
	tags []model.SkillTagSnapshot, aliases []string, explicit bool, restore func(model.SkillVersionSnapshot, model.SkillVersionID) error,
) error {
	_, err := tx.Skill(string(skill.ID))
	duplicateID := err == nil
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return err
	}
	if skill.SourceKey != "" {
		_, err := tx.SkillBySource(skill.SourceKey)
		switch {
		case err == nil:
			if !explicit {
				return ErrConflict
			}
			skill.SourceKey = ""
		case !errors.Is(err, repo.ErrNotFound):
			return err
		}
	}
	if !explicit {
		for _, key := range aliases {
			if _, err := tx.SkillBySource(key); err == nil {
				return ErrConflict
			} else if !errors.Is(err, repo.ErrNotFound) {
				return err
			}
		}
	}
	if duplicateID {
		if !explicit {
			return ErrConflict
		}
		skill.ID = model.SkillID(uuid.NewString())
	}

	if err := tx.CreateSkill(skill); err != nil {
		return err
	}
	for _, key := range aliases {
		if err := rememberSource(tx, string(skill.ID), key, explicit); err != nil {
			return err
		}
	}
	for _, tag := range tags {
		if duplicateID {
			tag.ID = model.SkillTagID(uuid.NewString())
		}
		tag.SkillId = string(skill.ID)
		if err := tx.CreateTag(tag); err != nil {
			return err
		}
	}
	for _, original := range versions {
		version := original
		version.SkillId = string(skill.ID)
		created, err := createImportedVersion(tx, version)
		if err != nil {
			return err
		}
		if skill.DefaultVersionId == string(original.ID) && created.ID != original.ID {
			skill.DefaultVersionId = string(created.ID)
			if err := tx.UpdateSkill(skill, "defaultVersionId"); err != nil {
				return err
			}
		}
		if err := restore(original, created.ID); err != nil {
			return err
		}
	}
	return nil
}
