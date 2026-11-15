package repo

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

type CatalogEntry struct {
	Skill   *model.Skill
	Version *model.SkillVersion
}

type CatalogRepo interface {
	AddSkills(context.Context, []CatalogEntry) (int, error)
	AddVersion(context.Context, *model.SkillVersion, string) (bool, error)
	Tags(context.Context, []model.SkillID) (map[model.SkillID][]string, error)
	UpdateMetadata(context.Context, model.SkillID, string, string, []string) (*model.Skill, error)
	DeleteSkill(context.Context, model.SkillID) error
}
