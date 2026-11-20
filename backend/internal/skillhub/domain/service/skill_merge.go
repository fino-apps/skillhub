package service

import (
	"context"
	"errors"
	"os"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

type VersionMergeSummary struct {
	Added, Restored, Skipped int32
}

func (s *VersionMergeSummary) add(other VersionMergeSummary) {
	s.Added += other.Added
	s.Restored += other.Restored
	s.Skipped += other.Skipped
}

type versionMergeStep struct {
	source, target model.SkillVersionSnapshot
	add, restore   bool
	targetAdded    bool
}

type versionMergePlan struct {
	steps   []versionMergeStep
	summary VersionMergeSummary
}

func matchVersionKey(version model.SkillVersionSnapshot) string {
	if version.ContentDigest != "" {
		return "digest:" + version.ContentDigest
	}
	return "id:" + string(version.ID) + ":" + version.CommitSha
}

func (s *Library) planVersionMerge(incoming, current []model.SkillVersionSnapshot, incomingAvailable func(model.SkillVersionSnapshot) bool) versionMergePlan {
	type match struct {
		row              model.SkillVersionSnapshot
		available, added bool
	}
	matches := make(map[string]match, len(current))
	for _, row := range current {
		key := matchVersionKey(row)
		available := s.Available(row)
		if _, exists := matches[key]; !exists || available {
			matches[key] = match{row: row, available: available}
		}
	}
	var plan versionMergePlan
	for _, row := range incoming {
		key := matchVersionKey(row)
		existing, found := matches[key]
		available := incomingAvailable(row)
		step := versionMergeStep{source: row, target: existing.row, targetAdded: existing.added}
		switch {
		case !found:
			step.add = true
			matches[key] = match{row: row, available: available, added: true}
			plan.summary.Added++
		case !existing.available && available:
			step.restore = true
			existing.available = true
			matches[key] = existing
			plan.summary.Restored++
		default:
			plan.summary.Skipped++
		}
		plan.steps = append(plan.steps, step)
	}
	return plan
}

func applyVersionMerge(plan versionMergePlan, source, target model.SkillSnapshot, tx repo.LibraryTx,
	add func(model.SkillVersionSnapshot) (model.SkillVersionSnapshot, error),
	restore func(model.SkillVersionSnapshot, model.SkillVersionID) error,
) error {
	ids := make(map[model.SkillVersionID]model.SkillVersionID, len(plan.steps))
	for _, step := range plan.steps {
		id := step.target.ID
		if step.add {
			version := step.source
			version.SkillId = string(target.ID)
			preserveVersionOrigin(&version, source)
			created, err := add(version)
			if err != nil {
				return err
			}
			ids[step.source.ID] = created.ID
			id = created.ID
		} else if step.targetAdded {
			id = ids[id]
		}
		if step.restore {
			if err := restore(step.source, id); err != nil {
				return err
			}
		}
		if target.DefaultVersionId == "" && source.DefaultVersionId == string(step.source.ID) {
			target.DefaultVersionId = string(id)
			if err := tx.UpdateSkill(target, "defaultVersionId"); err != nil {
				return err
			}
		}
	}
	return nil
}

func preserveVersionOrigin(version *model.SkillVersionSnapshot, source model.SkillSnapshot) {
	if version.UpstreamRepository == "" && version.CommitSha != "" && source.RepositoryUrl != "" {
		version.UpstreamRepository = source.RepositoryUrl
		version.UpstreamPath = source.RepositoryPath
		version.UpstreamCommit = version.CommitSha
	}
}

func (s *Library) MergeSkills(ctx context.Context, sourceID, targetID string, preview bool) (VersionMergeSummary, error) {
	var summary VersionMergeSummary
	if !canonicalBackupID(sourceID) || !canonicalBackupID(targetID) || sourceID == targetID {
		return summary, ErrInvalid
	}
	s.writes.Lock()
	defer s.writes.Unlock()

	var saved, removed []string
	err := s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		source, err := tx.Skill(sourceID)
		if err != nil {
			return err
		}
		target, err := tx.Skill(targetID)
		if err != nil {
			return err
		}
		incoming, err := tx.SkillVersions(sourceID)
		if err != nil {
			return err
		}
		current, err := tx.SkillVersions(targetID)
		if err != nil {
			return err
		}
		plan := s.planVersionMerge(incoming, current, s.Available)
		summary = plan.summary
		if preview {
			return nil
		}
		if err := applyVersionMerge(plan, source, target, tx,
			func(version model.SkillVersionSnapshot) (model.SkillVersionSnapshot, error) {
				return version, tx.MoveVersion(version)
			},
			func(version model.SkillVersionSnapshot, target model.SkillVersionID) error {
				data, err := s.ReadArchive(version)
				if err != nil {
					return err
				}
				if err := s.writeSnapshot(string(target), data); err != nil {
					return err
				}
				saved = append(saved, string(target))
				return nil
			}); err != nil {
			return err
		}
		removed, err = tx.DeleteSkill(source)
		return err
	})
	if err != nil {
		s.removeSnapshots(saved)
		return VersionMergeSummary{}, err
	}
	s.removeSnapshots(removed)
	return summary, nil
}

func (s *Library) removeSnapshots(ids []string) {
	for _, id := range ids {
		_ = os.Remove(s.snapshotPath(id))
	}
}

func createImportedVersion(tx repo.LibraryTx, version model.SkillVersionSnapshot) (model.SkillVersionSnapshot, error) {
	_, err := tx.Version("", string(version.ID))
	switch {
	case err == nil:
		version.ID = model.SkillVersionID(uuid.NewString())
	case !errors.Is(err, repo.ErrNotFound):
		return version, err
	}
	return version, tx.CreateVersion(version)
}
