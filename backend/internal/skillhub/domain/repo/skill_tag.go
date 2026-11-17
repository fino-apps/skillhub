package repo

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

//go:generate mockgen -destination=mocks/skill_tag.go -package=mocks . ISkillTagRepo
type ISkillTagRepo interface {
	GetSkillTag(ctx context.Context, id model.SkillTagID) (*model.SkillTag, error)
	ListSkillTags(ctx context.Context, query model.ListSkillTagQuery) (model.ListSkillTagResult, error)
	CreateSkillTag(ctx context.Context, entity *model.SkillTag) (*model.SkillTag, error)
	UpdateSkillTag(ctx context.Context, entity *model.SkillTag, updateMask []string) error
	DeleteSkillTag(ctx context.Context, id model.SkillTagID) error
}
