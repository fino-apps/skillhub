package repo

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

//go:generate mockgen -destination=mocks/skill_source_alias.go -package=mocks . ISkillSourceAliasRepo
type ISkillSourceAliasRepo interface {
	GetSkillSourceAlias(ctx context.Context, id model.SkillSourceAliasID) (*model.SkillSourceAlias, error)
	ListSkillSourceAliass(ctx context.Context, query model.ListSkillSourceAliasQuery) (model.ListSkillSourceAliasResult, error)
	CreateSkillSourceAlias(ctx context.Context, entity *model.SkillSourceAlias) (*model.SkillSourceAlias, error)
	UpdateSkillSourceAlias(ctx context.Context, entity *model.SkillSourceAlias, updateMask []string) error
	DeleteSkillSourceAlias(ctx context.Context, id model.SkillSourceAliasID) error
}
