package app

import (
	"context"
	"testing"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

type installService struct {
	service.ISkillhubService
	skill   *model.Skill
	version *model.SkillVersion
}

func (s installService) GetSkill(context.Context, model.SkillID) (*model.Skill, error) {
	return s.skill, nil
}
func (s installService) GetSkillVersion(context.Context, model.SkillVersionID) (*model.SkillVersion, error) {
	return s.version, nil
}

func TestInstallCommandsRejectsVersionFromAnotherSkill(t *testing.T) {
	srv := skillhubServer{skillhubSvc: installService{
		skill:   model.RebuildSkill(model.SkillSnapshot{ID: "one", Name: "sample"}),
		version: model.RebuildSkillVersion(model.SkillVersionSnapshot{SkillId: "other"}),
	}}
	_, err := srv.getInstallCommand(context.Background(), &pb.GetInstallCommandRequest{Global: true, SkillId: "one", VersionId: "v1"})
	if err == nil {
		t.Fatal("expected mismatched version to be rejected")
	}
}
