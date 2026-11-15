package app

import (
	"context"
	"errors"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) importLocalSkills(ctx context.Context, paths []string) (*pb.ImportSkillResponse, error) {
	entries := make([]repo.CatalogEntry, 0, len(paths))
	pending := make(map[model.SkillID]*service.SkillSnapshot, len(paths))
	defer func() {
		for _, snapshot := range pending {
			snapshot.Discard()
		}
	}()
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(path) > 2048 {
			return nil, errcode.NewPlatformBadRequest()
		}
		snapshot, err := s.snapshots.Capture(path, "")
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		if seen[snapshot.Path] {
			snapshot.Discard()
			continue
		}
		seen[snapshot.Path] = true
		skill, err := model.NewSkill(model.CreateSkillInput{
			Name: snapshot.Name, Description: snapshot.Description, LocalPath: snapshot.Path,
		})
		if err != nil {
			snapshot.Discard()
			return nil, errcode.WrapCreateSkillhubFailed(err)
		}
		pending[skill.ID()] = snapshot
		version, err := snapshotVersion(skill.ID(), snapshot, "")
		if err != nil {
			return nil, errcode.WrapCreateSkillhubFailed(err)
		}
		if err := s.snapshots.Publish(snapshot, string(skill.ID()), string(version.ID())); err != nil {
			return nil, errcode.WrapCreateSkillhubFailed(err)
		}
		entries = append(entries, repo.CatalogEntry{Skill: skill, Version: version})
	}
	added, err := s.catalog.AddSkills(ctx, entries)
	if err != nil {
		return nil, errcode.WrapCreateSkillhubFailed(err)
	}
	for _, id := range added {
		delete(pending, id)
	}
	return &pb.ImportSkillResponse{AddedCount: int32(len(added)), ExistingCount: int32(len(entries) - len(added))}, nil
}

func (s skillhubServer) captureVersion(ctx context.Context, skill *model.Skill, input *pb.VersionInput) (*pb.CreateVersionResponse, error) {
	snapshot, err := s.snapshots.Capture(skill.LocalPath(), input.GetExpectedDigest())
	if errors.Is(err, service.ErrSnapshotChanged) {
		return nil, errcode.WrapPlatformConflict(err)
	}
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}

	saved := false
	defer func() {
		if !saved {
			snapshot.Discard()
		}
	}()
	version, err := snapshotVersion(skill.ID(), snapshot, input.GetNotes())
	if err != nil {
		return nil, errcode.WrapCreateSkillhubFailed(err)
	}
	if err := s.snapshots.Publish(snapshot, string(skill.ID()), string(version.ID())); err != nil {
		return nil, errcode.WrapCreateSkillhubFailed(err)
	}
	result, err := s.saveVersion(ctx, version, snapshot.Description)
	saved = err == nil
	return result, err
}

func snapshotVersion(id model.SkillID, snapshot *service.SkillSnapshot, notes string) (*model.SkillVersion, error) {
	return model.NewSkillVersion(model.CreateSkillVersionInput{
		SkillId: string(id), Name: snapshot.Name, Notes: notes, ContentDigest: snapshot.Digest,
	})
}
