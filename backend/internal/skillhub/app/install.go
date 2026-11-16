package app

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) getInstallCommand(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	path, err := s.versionPath(version)
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	command, err := service.LocalInstallCommand(path, version.Name(), in.GetGlobal())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.GetInstallCommandResponse{Command: command}, nil
}
