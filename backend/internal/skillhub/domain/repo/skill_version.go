package repo

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

//go:generate mockgen -destination=mocks/skill_version.go -package=mocks . ISkillVersionRepo
type ISkillVersionRepo interface {
	GetSkillVersion(ctx context.Context, id model.SkillVersionID) (*model.SkillVersion, error)
	ListSkillVersions(ctx context.Context, query model.ListSkillVersionQuery) (model.ListSkillVersionResult, error)
	CreateSkillVersion(ctx context.Context, entity *model.SkillVersion) (*model.SkillVersion, error)
	UpdateSkillVersion(ctx context.Context, entity *model.SkillVersion, updateMask []string) error
	DeleteSkillVersion(ctx context.Context, id model.SkillVersionID) error
}
