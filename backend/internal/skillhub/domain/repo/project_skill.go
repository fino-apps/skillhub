package repo

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

//go:generate mockgen -destination=mocks/project_skill.go -package=mocks . IProjectSkillRepo
type IProjectSkillRepo interface {
	GetProjectSkill(ctx context.Context, id model.ProjectSkillID) (*model.ProjectSkill, error)
	ListProjectSkills(ctx context.Context, query model.ListProjectSkillQuery) (model.ListProjectSkillResult, error)
	CreateProjectSkill(ctx context.Context, entity *model.ProjectSkill) (*model.ProjectSkill, error)
	UpdateProjectSkill(ctx context.Context, entity *model.ProjectSkill, updateMask []string) error
	DeleteProjectSkill(ctx context.Context, id model.ProjectSkillID) error
}
