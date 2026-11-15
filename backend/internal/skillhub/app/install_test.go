package app

import (
	"context"
	"os"
	"path/filepath"
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
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "snapshot"), 0700); err != nil {
		t.Fatal(err)
	}
	srv := skillhubServer{skillhubSvc: installService{skill: model.RebuildSkill(model.SkillSnapshot{ID: "one", Name: "sample"}), version: model.RebuildSkillVersion(model.SkillVersionSnapshot{SkillId: "other", SnapshotPath: "snapshot"})}, intake: service.NewIntake(root, nil)}
	_, err := srv.getInstallCommands(context.Background(), &pb.GetInstallCommandsRequest{Global: true, Skills: []*pb.InstallSelection{{SkillId: "one", VersionId: "v1"}}})
	if err == nil {
		t.Fatal("expected mismatched version to be rejected")
	}
}
