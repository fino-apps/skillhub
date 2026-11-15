package app

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) getInstallCommands(ctx context.Context, in *pb.GetInstallCommandsRequest) (*pb.GetInstallCommandsResponse, error) {
	if s.intake == nil || len(in.GetSkills()) == 0 {
		return nil, errcode.NewPlatformBadRequest()
	}
	commands := make([]string, 0, len(in.GetSkills()))
	seen := make(map[string]bool)
	for _, selection := range in.GetSkills() {
		if selection == nil || selection.GetSkillId() == "" || selection.GetVersionId() == "" || seen[selection.GetSkillId()] {
			return nil, errcode.NewPlatformBadRequest()
		}
		seen[selection.GetSkillId()] = true
		skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(selection.GetSkillId()))
		if err != nil {
			return nil, errcode.WrapPlatformNotFound(err)
		}
		version, err := s.skillhubSvc.GetSkillVersion(ctx, model.SkillVersionID(selection.GetVersionId()))
		if err != nil {
			return nil, errcode.WrapPlatformNotFound(err)
		}
		if version.SkillId() != selection.GetSkillId() {
			return nil, errcode.NewPlatformBadRequest()
		}
		var snapshot string
		if version.SnapshotPath() != "" {
			snapshot, err = s.intake.StoredPath(version.SnapshotPath())
			if err != nil {
				return nil, errcode.WrapPlatformBadRequest(err)
			}
		}
		command, err := service.InstallCommand(snapshot, skill.RepositoryUrl(), version.CommitSha(), skill.Name(), in.GetProjectPath(), in.GetGlobal())
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		commands = append(commands, command)
	}
	return &pb.GetInstallCommandsResponse{Commands: commands}, nil
}
