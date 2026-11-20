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
	id       model.SkillVersionID
	imported bool
}

type versionMergeStep struct {
	source  model.SkillVersionSnapshot
	target  mergeTarget
	add     bool
	repairs []mergeTarget
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

func planVersionMerge(incoming, current []model.SkillVersionSnapshot, defaultID string,
	incomingUsable, currentUsable func(model.SkillVersionSnapshot) bool,
) versionMergePlan {
	type match struct {
		target mergeTarget
		usable bool
	}
	matches := make(map[string][]*match, len(current))
	// Only matching versions can affect a repair or cause a source snapshot to be discarded.
	keys := make(map[string]bool, len(incoming))
	for _, row := range incoming {
		keys[matchVersionKey(row)] = true
	}
	for _, row := range current {
		key := matchVersionKey(row)
		if keys[key] {
			matches[key] = append(matches[key], &match{target: mergeTarget{id: row.ID}, usable: currentUsable(row)})
		}
	}

	var plan versionMergePlan
	for _, row := range incoming {
		key := matchVersionKey(row)
		options := matches[key]
		step := versionMergeStep{source: row}
		usable := incomingUsable(row)
		if len(options) == 0 {
			step.add = true
			matches[key] = []*match{{target: mergeTarget{id: row.ID, imported: true}, usable: usable}}
			plan.summary.Added++
		} else {
			preferred := options[0]
			hasUsable := false
			for _, option := range options {
				id := string(option.target.id)
				if option.target.id == row.ID || id == defaultID && preferred.target.id != row.ID {
					preferred = option
				}
				hasUsable = hasUsable || option.usable
				if usable && !option.usable && (option.target.id == row.ID || id == defaultID) {
					step.repairs = append(step.repairs, option.target)
					option.usable = true
				}
			}
			step.target = preferred.target
			if usable && !hasUsable && len(step.repairs) == 0 {
				step.repairs = append(step.repairs, preferred.target)
				preferred.usable = true
			}
			plan.summary.Restored += int32(len(step.repairs))
			if len(step.repairs) == 0 {
				plan.summary.Skipped++
			}
		}
		plan.steps = append(plan.steps, step)
	}
	return plan
}

func (p versionMergePlan) details(source model.SkillSnapshot) VersionMergeSummary {
	summary := p.summary
	for _, step := range p.steps {
		change := VersionMergeChange{Version: step.source, Action: "skip"}
		change.Source = cmp.Or(step.source.UpstreamRepository, mergeEntry(source, nil).Source)
		if step.source.UpstreamPath != "" {
			change.Source += " · " + step.source.UpstreamPath
		}
		if step.add {
			change.Action = "add"
		} else if len(step.repairs) > 0 {
			change.Action = "restore"
			for _, repair := range step.repairs {
				change.Restored = append(change.Restored, string(repair.id))
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
		if target.imported {
			return ids[target.id]
		}
		return target.id
	}
	for _, step := range plan.steps {
		id := resolve(step.target)
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
		}
		for _, repaired := range step.repairs {
			if err := restore(step.source, resolve(repaired)); err != nil {
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

func mergeRevision(source, target model.SkillSnapshot, incoming, current []model.SkillVersionSnapshot, summary VersionMergeSummary) (string, error) {
	hash := sha256.New()
	input := struct {
		Source, Target    model.SkillSnapshot
		Incoming, Current []model.SkillVersionSnapshot
		Summary           VersionMergeSummary
	}{source, target, incoming, current, summary}
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
		summary = plan.details(source)
		summary.Revision, err = mergeRevision(source, target, incoming, current, summary)
		if err != nil || revision == "" {
			return err
		}
		if revision != summary.Revision {
			return fmt.Errorf("%w: Skill 已变化，请重新预览合并", ErrConflict)
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
