package repo

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

//go:generate mockgen -destination=mocks/skill.go -package=mocks . ISkillRepo
type ISkillRepo interface {
	GetSkill(ctx context.Context, id model.SkillID) (*model.Skill, error)
	ListSkills(ctx context.Context, query model.ListSkillQuery) (model.ListSkillResult, error)
	CreateSkill(ctx context.Context, entity *model.Skill) (*model.Skill, error)
	UpdateSkill(ctx context.Context, entity *model.Skill, updateMask []string) error
	DeleteSkill(ctx context.Context, id model.SkillID) error
}
