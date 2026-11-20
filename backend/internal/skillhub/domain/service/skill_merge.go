package service

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

type VersionMergeSummary struct {
	Added, Restored, Skipped int32
	Revision                 string
	Changes                  []VersionMergeChange
}

type VersionMergeChange struct {
	Version  model.SkillVersionSnapshot
	Source   string
	Action   string
	Restored []string
}

func (s *VersionMergeSummary) add(other VersionMergeSummary) {
	s.Added += other.Added
	s.Restored += other.Restored
	s.Skipped += other.Skipped
}

type mergeTarget struct {
	ID       model.SkillVersionID
	Imported bool
}

type versionMergeStep struct {
	Source       model.SkillVersionSnapshot
	SourceUsable bool
	Target       mergeTarget
	Add          bool
	Repairs      []mergeTarget
}

type versionMergePlan struct {
	steps           []versionMergeStep
	targetSnapshots map[model.SkillVersionID]bool
	summary         VersionMergeSummary
}

func matchVersionKey(version model.SkillVersionSnapshot) string {
	if version.ContentDigest != "" {
		return "digest:" + version.ContentDigest
	}
	return "id:" + string(version.ID) + ":" + version.CommitSha
}

func planVersionMerge(incoming, current []model.SkillVersionSnapshot, defaultID string,
	incomingUsable, currentUsable func(model.SkillVersionSnapshot) bool,
) versionMergePlan {
	type match struct {
		target mergeTarget
		usable bool
	}

	plan := versionMergePlan{targetSnapshots: make(map[model.SkillVersionID]bool)}
	matches := make(map[string][]match, len(incoming))
	for _, row := range incoming {
		matches[matchVersionKey(row)] = nil
	}
	// Only existing versions that can match an incoming version need snapshot validation.
	for _, row := range current {
		key := matchVersionKey(row)
		if options, relevant := matches[key]; relevant {
			usable := currentUsable(row)
			matches[key] = append(options, match{target: mergeTarget{ID: row.ID}, usable: usable})
			plan.targetSnapshots[row.ID] = usable
		}
	}

	for _, row := range incoming {
		key := matchVersionKey(row)
		options := matches[key]
		step := versionMergeStep{Source: row, SourceUsable: incomingUsable(row)}
		if len(options) == 0 {
			step.Add = true
			matches[key] = []match{{target: mergeTarget{ID: row.ID, Imported: true}, usable: step.SourceUsable}}
			plan.summary.Added++
			plan.steps = append(plan.steps, step)
			continue
		}

		preferred := &options[0]
		hasUsable := false
		for i := range options {
			option := &options[i]
			exact := option.target.ID == row.ID
			isDefault := string(option.target.ID) == defaultID
			if exact || isDefault && preferred.target.ID != row.ID {
				preferred = option
			}
			hasUsable = hasUsable || option.usable
			if step.SourceUsable && !option.usable && (exact || isDefault) {
				step.Repairs = append(step.Repairs, option.target)
				option.usable = true
			}
		}
		step.Target = preferred.target
		if step.SourceUsable && !hasUsable && len(step.Repairs) == 0 {
			step.Repairs = append(step.Repairs, preferred.target)
			preferred.usable = true
		}
		plan.summary.Restored += int32(len(step.Repairs))
		if len(step.Repairs) == 0 {
			plan.summary.Skipped++
		}
		plan.steps = append(plan.steps, step)
	}
	return plan
}

func (p versionMergePlan) details(source model.SkillSnapshot) VersionMergeSummary {
	summary := p.summary
	sourceLocation := mergeEntry(source, nil).Source
	for _, step := range p.steps {
		change := VersionMergeChange{Version: step.Source, Action: "skip"}
		change.Source = cmp.Or(step.Source.UpstreamRepository, sourceLocation)
		if step.Source.UpstreamPath != "" {
			change.Source += " · " + step.Source.UpstreamPath
		}
		if step.Add {
			change.Action = "add"
		} else if len(step.Repairs) > 0 {
			change.Action = "restore"
			for _, repair := range step.Repairs {
				change.Restored = append(change.Restored, string(repair.ID))
			}
		}
		summary.Changes = append(summary.Changes, change)
	}
	return summary
}

func applyVersionMerge(plan versionMergePlan, source, target model.SkillSnapshot, tx repo.LibraryTx,
	add func(model.SkillVersionSnapshot) (model.SkillVersionSnapshot, error),
	restore func(model.SkillVersionSnapshot, model.SkillVersionID) error,
) error {
	ids := make(map[model.SkillVersionID]model.SkillVersionID, len(plan.steps))
	resolve := func(target mergeTarget) model.SkillVersionID {
		if target.Imported {
			return ids[target.ID]
		}
		return target.ID
	}
	for _, step := range plan.steps {
		id := resolve(step.Target)
		if step.Add {
			version := step.Source
			version.SkillId = string(target.ID)
			preserveVersionOrigin(&version, source)
			created, err := add(version)
			if err != nil {
				return err
			}
			ids[step.Source.ID] = created.ID
			id = created.ID
		}
		for _, repaired := range step.Repairs {
			if err := restore(step.Source, resolve(repaired)); err != nil {
				return err
			}
		}
		if target.DefaultVersionId == "" && source.DefaultVersionId == string(step.Source.ID) {
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

func (p versionMergePlan) revision(source, target model.SkillSnapshot, current []model.SkillVersionSnapshot) (string, error) {
	hash := sha256.New()
	input := struct {
		Source, Target  model.SkillSnapshot
		CurrentVersions []model.SkillVersionSnapshot
		TargetSnapshots map[model.SkillVersionID]bool
		Steps           []versionMergeStep
	}{
		Source: source, Target: target, CurrentVersions: current,
		TargetSnapshots: p.targetSnapshots, Steps: p.steps,
	}
	if err := json.NewEncoder(hash).Encode(input); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (s *Library) MergeSkills(ctx context.Context, sourceID, targetID, revision string) (VersionMergeSummary, error) {
	var summary VersionMergeSummary
	if !canonicalBackupID(sourceID) || !canonicalBackupID(targetID) || sourceID == targetID {
		return summary, ErrInvalid
	}
	s.writes.Lock()
	defer s.writes.Unlock()

	changes := snapshotChanges{library: s}
	var removed []string
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
		usable := s.snapshotChecks()
		plan := planVersionMerge(incoming, current, target.DefaultVersionId, usable, usable)
		currentRevision, err := plan.revision(source, target, current)
		if err != nil {
			return err
		}
		if revision != "" && revision != currentRevision {
			return fmt.Errorf("%w: Skill 已变化，请重新预览合并", ErrConflict)
		}
		summary = plan.details(source)
		summary.Revision = currentRevision
		if revision == "" {
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
				if err := validateBackupSnapshot(data, version.ContentDigest); err != nil {
					return invalid(err)
				}
				return changes.write(target, data)
			}); err != nil {
			return err
		}
		if err := tx.MoveSourceAliases(sourceID, targetID); err != nil {
			return err
		}
		removed, err = tx.DeleteSkill(source)
		if err != nil {
			return err
		}
		return rememberSource(tx, targetID, source.SourceKey, false)
	})
	if err = changes.finish(err); err != nil {
		return VersionMergeSummary{}, err
	}
	for _, id := range removed {
		_ = os.Remove(s.snapshotPath(id))
	}
	return summary, nil
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
