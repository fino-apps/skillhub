package service

import (
	"context"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
)

//go:generate mockgen -destination=mocks/skillhub_service.go -package=mocks . ISkillhubService
type ISkillhubService interface {
	GetProjectSkill(ctx context.Context, id model.ProjectSkillID) (*model.ProjectSkill, error)
	ListProjectSkills(ctx context.Context, query model.ListProjectSkillQuery) (model.ListProjectSkillResult, error)
	CreateProjectSkill(ctx context.Context, input model.CreateProjectSkillInput) (*model.ProjectSkill, error)
	UpdateProjectSkill(ctx context.Context, id model.ProjectSkillID, input model.UpdateProjectSkillInput) (*model.ProjectSkill, error)
	DeleteProjectSkill(ctx context.Context, id model.ProjectSkillID) error

	GetSkill(ctx context.Context, id model.SkillID) (*model.Skill, error)
	ListSkills(ctx context.Context, query model.ListSkillQuery) (model.ListSkillResult, error)
	CreateSkill(ctx context.Context, input model.CreateSkillInput) (*model.Skill, error)
	UpdateSkill(ctx context.Context, id model.SkillID, input model.UpdateSkillInput) (*model.Skill, error)
	DeleteSkill(ctx context.Context, id model.SkillID) error

	GetSkillTag(ctx context.Context, id model.SkillTagID) (*model.SkillTag, error)
	ListSkillTags(ctx context.Context, query model.ListSkillTagQuery) (model.ListSkillTagResult, error)
	CreateSkillTag(ctx context.Context, input model.CreateSkillTagInput) (*model.SkillTag, error)
	UpdateSkillTag(ctx context.Context, id model.SkillTagID, input model.UpdateSkillTagInput) (*model.SkillTag, error)
	DeleteSkillTag(ctx context.Context, id model.SkillTagID) error

	GetSkillVersion(ctx context.Context, id model.SkillVersionID) (*model.SkillVersion, error)
	ListSkillVersions(ctx context.Context, query model.ListSkillVersionQuery) (model.ListSkillVersionResult, error)
	CreateSkillVersion(ctx context.Context, input model.CreateSkillVersionInput) (*model.SkillVersion, error)
	UpdateSkillVersion(ctx context.Context, id model.SkillVersionID, input model.UpdateSkillVersionInput) (*model.SkillVersion, error)
	DeleteSkillVersion(ctx context.Context, id model.SkillVersionID) error
}

type ServiceDeps struct {
	ProjectSkillRepo repo.IProjectSkillRepo
	SkillRepo        repo.ISkillRepo
	SkillTagRepo     repo.ISkillTagRepo
	SkillVersionRepo repo.ISkillVersionRepo
}

type SkillhubServiceImpl struct {
	projectSkillRepo repo.IProjectSkillRepo
	skillRepo        repo.ISkillRepo
	skillTagRepo     repo.ISkillTagRepo
	skillVersionRepo repo.ISkillVersionRepo
}

func NewSkillhubService(deps ServiceDeps) ISkillhubService {
	return &SkillhubServiceImpl{
		projectSkillRepo: deps.ProjectSkillRepo,
		skillRepo:        deps.SkillRepo,
		skillTagRepo:     deps.SkillTagRepo,
		skillVersionRepo: deps.SkillVersionRepo,
	}
}

func (s *SkillhubServiceImpl) GetProjectSkill(ctx context.Context, id model.ProjectSkillID) (*model.ProjectSkill, error) {
	return s.projectSkillRepo.GetProjectSkill(ctx, id)
}

func (s *SkillhubServiceImpl) ListProjectSkills(ctx context.Context, query model.ListProjectSkillQuery) (model.ListProjectSkillResult, error) {
	return s.projectSkillRepo.ListProjectSkills(ctx, query)
}

func (s *SkillhubServiceImpl) CreateProjectSkill(ctx context.Context, input model.CreateProjectSkillInput) (*model.ProjectSkill, error) {
	// TODO: set ProjectPath, SkillPath, SkillId, BaseVersionId, BaseDigest before creating ProjectSkill.
	entity, err := model.NewProjectSkill(input)
	if err != nil {
		return nil, err
	}

	return s.projectSkillRepo.CreateProjectSkill(ctx, entity)
}

func (s *SkillhubServiceImpl) UpdateProjectSkill(ctx context.Context, id model.ProjectSkillID, input model.UpdateProjectSkillInput) (*model.ProjectSkill, error) {
	entity, err := s.projectSkillRepo.GetProjectSkill(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(input.UpdateMask) > 0 {
		input.UpdateTime = core.Now()
		input.UpdateMask = append(input.UpdateMask, "updateTime")
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.projectSkillRepo.UpdateProjectSkill(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *SkillhubServiceImpl) DeleteProjectSkill(ctx context.Context, id model.ProjectSkillID) error {
	return s.projectSkillRepo.DeleteProjectSkill(ctx, id)
}

func (s *SkillhubServiceImpl) GetSkill(ctx context.Context, id model.SkillID) (*model.Skill, error) {
	return s.skillRepo.GetSkill(ctx, id)
}

func (s *SkillhubServiceImpl) ListSkills(ctx context.Context, query model.ListSkillQuery) (model.ListSkillResult, error) {
	return s.skillRepo.ListSkills(ctx, query)
}

func (s *SkillhubServiceImpl) CreateSkill(ctx context.Context, input model.CreateSkillInput) (*model.Skill, error) {
	// TODO: set ContentPath, CurrentDigest, OriginCommitSha before creating Skill.
	entity, err := model.NewSkill(input)
	if err != nil {
		return nil, err
	}

	return s.skillRepo.CreateSkill(ctx, entity)
}

func (s *SkillhubServiceImpl) UpdateSkill(ctx context.Context, id model.SkillID, input model.UpdateSkillInput) (*model.Skill, error) {
	entity, err := s.skillRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(input.UpdateMask) > 0 {
		input.UpdateTime = core.Now()
		input.UpdateMask = append(input.UpdateMask, "updateTime")
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.skillRepo.UpdateSkill(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *SkillhubServiceImpl) DeleteSkill(ctx context.Context, id model.SkillID) error {
	return s.skillRepo.DeleteSkill(ctx, id)
}

func (s *SkillhubServiceImpl) GetSkillTag(ctx context.Context, id model.SkillTagID) (*model.SkillTag, error) {
	return s.skillTagRepo.GetSkillTag(ctx, id)
}

func (s *SkillhubServiceImpl) ListSkillTags(ctx context.Context, query model.ListSkillTagQuery) (model.ListSkillTagResult, error) {
	return s.skillTagRepo.ListSkillTags(ctx, query)
}

func (s *SkillhubServiceImpl) CreateSkillTag(ctx context.Context, input model.CreateSkillTagInput) (*model.SkillTag, error) {
	// TODO: set SkillId, Tag before creating SkillTag.
	entity, err := model.NewSkillTag(input)
	if err != nil {
		return nil, err
	}

	return s.skillTagRepo.CreateSkillTag(ctx, entity)
}

func (s *SkillhubServiceImpl) UpdateSkillTag(ctx context.Context, id model.SkillTagID, input model.UpdateSkillTagInput) (*model.SkillTag, error) {
	entity, err := s.skillTagRepo.GetSkillTag(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(input.UpdateMask) > 0 {
		input.UpdateTime = core.Now()
		input.UpdateMask = append(input.UpdateMask, "updateTime")
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.skillTagRepo.UpdateSkillTag(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *SkillhubServiceImpl) DeleteSkillTag(ctx context.Context, id model.SkillTagID) error {
	return s.skillTagRepo.DeleteSkillTag(ctx, id)
}

func (s *SkillhubServiceImpl) GetSkillVersion(ctx context.Context, id model.SkillVersionID) (*model.SkillVersion, error) {
	return s.skillVersionRepo.GetSkillVersion(ctx, id)
}

func (s *SkillhubServiceImpl) ListSkillVersions(ctx context.Context, query model.ListSkillVersionQuery) (model.ListSkillVersionResult, error) {
	return s.skillVersionRepo.ListSkillVersions(ctx, query)
}

func (s *SkillhubServiceImpl) CreateSkillVersion(ctx context.Context, input model.CreateSkillVersionInput) (*model.SkillVersion, error) {
	// TODO: set SkillId, Version, Notes, Digest, SnapshotPath, SourceRef, CommitSha before creating SkillVersion.
	entity, err := model.NewSkillVersion(input)
	if err != nil {
		return nil, err
	}

	return s.skillVersionRepo.CreateSkillVersion(ctx, entity)
}

func (s *SkillhubServiceImpl) UpdateSkillVersion(ctx context.Context, id model.SkillVersionID, input model.UpdateSkillVersionInput) (*model.SkillVersion, error) {
	entity, err := s.skillVersionRepo.GetSkillVersion(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(input.UpdateMask) > 0 {
		input.UpdateTime = core.Now()
		input.UpdateMask = append(input.UpdateMask, "updateTime")
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.skillVersionRepo.UpdateSkillVersion(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *SkillhubServiceImpl) DeleteSkillVersion(ctx context.Context, id model.SkillVersionID) error {
	return s.skillVersionRepo.DeleteSkillVersion(ctx, id)
}
