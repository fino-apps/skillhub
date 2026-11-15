package repo

import (
	"context"

	"github.com/fino-io/finokit/db"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	domainrepo "github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/mapper"
	pmodel "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/model/sqlite"
	"gorm.io/gorm"
)

type IntakeRepo struct{ provider db.Provider }

func NewIntakeRepo(provider db.Provider) *IntakeRepo { return &IntakeRepo{provider: provider} }

func (r *IntakeRepo) CreateInitial(ctx context.Context, skill *model.Skill, version *model.SkillVersion, publish func() error) error {
	return r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		query := tx.Model(&pmodel.Skill{}).Where("kind = ?", skill.Kind())
		if skill.RepositoryUrl() != "" {
			query = query.Where("repository_url = ? AND repository_path = ?", skill.RepositoryUrl(), skill.RepositoryPath())
		} else {
			query = query.Where("name = ?", skill.Name())
		}
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return domainrepo.ErrSkillExists
		}
		if err := publish(); err != nil {
			return err
		}
		if err := tx.Create(mapper.SkillToRecord(skill)).Error; err != nil {
			return err
		}
		return tx.Create(mapper.SkillVersionToRecord(version)).Error
	})
}
