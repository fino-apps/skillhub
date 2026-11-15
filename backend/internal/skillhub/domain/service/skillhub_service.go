package service

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
)

type ISkillhubService interface {
	GetSkill(context.Context, model.SkillID) (*model.Skill, error)
	ListSkills(context.Context, model.ListSkillQuery) (model.ListSkillResult, error)
	GetSkillVersion(context.Context, model.SkillVersionID) (*model.SkillVersion, error)
	ListSkillVersions(context.Context, model.ListSkillVersionQuery) (model.ListSkillVersionResult, error)
}

type ServiceDeps struct {
	SkillRepo        repo.ISkillRepo
	SkillVersionRepo repo.ISkillVersionRepo
}

type SkillhubServiceImpl struct {
	skills   repo.ISkillRepo
	versions repo.ISkillVersionRepo
}

func NewSkillhubService(deps ServiceDeps) ISkillhubService {
	return &SkillhubServiceImpl{skills: deps.SkillRepo, versions: deps.SkillVersionRepo}
}

func (s *SkillhubServiceImpl) GetSkill(ctx context.Context, id model.SkillID) (*model.Skill, error) {
	return s.skills.GetSkill(ctx, id)
}

func (s *SkillhubServiceImpl) ListSkills(ctx context.Context, query model.ListSkillQuery) (model.ListSkillResult, error) {
	return s.skills.ListSkills(ctx, query)
}

func (s *SkillhubServiceImpl) GetSkillVersion(ctx context.Context, id model.SkillVersionID) (*model.SkillVersion, error) {
	return s.versions.GetSkillVersion(ctx, id)
}

func (s *SkillhubServiceImpl) ListSkillVersions(ctx context.Context, query model.ListSkillVersionQuery) (model.ListSkillVersionResult, error) {
	return s.versions.ListSkillVersions(ctx, query)
}
