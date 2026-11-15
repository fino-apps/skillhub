package app

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) getCopyCommand(ctx context.Context, in *pb.GetCopyCommandRequest) (*pb.GetCopyCommandResponse, error) {
	skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId()))
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	version, err := s.skillhubSvc.GetSkillVersion(ctx, model.SkillVersionID(in.GetVersionId()))
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	if version.SkillId() != in.GetSkillId() {
		return nil, errcode.NewPlatformBadRequest()
	}
	var snapshot string
	if version.SnapshotPath() != "" {
		if s.intake == nil {
			return nil, errcode.NewPlatformBadRequest()
		}
		snapshot, err = s.intake.StoredPath(version.SnapshotPath())
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
	}
	command, err := service.CopyCommand(snapshot, skill.RepositoryUrl(), skill.RepositoryPath(), version.CommitSha(), skill.Name())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.GetCopyCommandResponse{Command: command}, nil
}
