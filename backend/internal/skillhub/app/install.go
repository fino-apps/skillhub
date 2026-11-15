package app

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) getInstallCommand(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
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
	var command string
	if version.ContentDigest() != "" {
		path, pathErr := s.snapshots.Path(version.SkillId(), string(version.ID()), version.Name())
		if pathErr != nil {
			return nil, errcode.WrapPlatformNotFound(pathErr)
		}
		command, err = service.LocalInstallCommand(path, version.Name(), in.GetGlobal())
	} else {
		command, err = service.InstallCommand(skill.RepositoryUrl(), skill.RepositoryPath(), version.CommitSha(), version.Name(), in.GetGlobal())
	}
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.GetInstallCommandResponse{Command: command}, nil
}
