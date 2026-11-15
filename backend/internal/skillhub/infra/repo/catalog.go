package repo

import (
	"context"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/db"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	domainrepo "github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/mapper"
	pmodel "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/model/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CatalogRepo struct{ provider db.Provider }

func NewCatalogRepo(provider db.Provider) *CatalogRepo { return &CatalogRepo{provider: provider} }

func (r *CatalogRepo) AddSkills(ctx context.Context, entries []domainrepo.CatalogEntry) (int, error) {
	added := 0
	err := r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		for _, entry := range entries {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(mapper.SkillToRecord(entry.Skill))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}
			if err := tx.Create(mapper.SkillVersionToRecord(entry.Version)).Error; err != nil {
				return err
			}
			added++
		}
		return nil
	})
	return added, err
}

func (r *CatalogRepo) AddVersion(ctx context.Context, version *model.SkillVersion, description string) (bool, error) {
	var created bool
	err := r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		var skill pmodel.Skill
		if err := tx.Where("id = ?", version.SkillId()).Take(&skill).Error; err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(mapper.SkillVersionToRecord(version))
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		created = true
		return tx.Model(&skill).Updates(map[string]any{
			"name": version.Name(), "description": description, "update_time": core.Now(),
		}).Error
	})
	return created, err
}

func (r *CatalogRepo) DeleteSkill(ctx context.Context, id model.SkillID) error {
	return r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_id = ?", id).Delete(&pmodel.SkillVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Where("skill_id = ?", id).Delete(&pmodel.SkillTag{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&pmodel.Skill{})
		if result.Error == nil && result.RowsAffected == 0 {
			return core.NewNotFoundError("skill %s", id)
		}
		return result.Error
	})
}

func (r *CatalogRepo) Tags(ctx context.Context, ids []model.SkillID) (map[model.SkillID][]string, error) {
	tags := make(map[model.SkillID][]string, len(ids))
	if len(ids) == 0 {
		return tags, nil
	}
	var records []pmodel.SkillTag
	if err := r.provider.NewSession(ctx).Where("skill_id IN ?", ids).Order("tag").Find(&records).Error; err != nil {
		return nil, err
	}
	for _, record := range records {
		id := model.SkillID(record.SkillId)
		tags[id] = append(tags[id], record.Tag)
	}
	return tags, nil
}

func (r *CatalogRepo) UpdateMetadata(ctx context.Context, id model.SkillID, category, notes string, tags []string) (*model.Skill, error) {
	var skill pmodel.Skill
	err := r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&pmodel.Skill{}).Where("id = ?", id).Updates(map[string]any{
			"category": category, "notes": notes, "update_time": core.Now(),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return core.NewNotFoundError("skill %s", id)
		}
		if err := tx.Where("skill_id = ?", id).Delete(&pmodel.SkillTag{}).Error; err != nil {
			return err
		}
		for _, tag := range tags {
			label, err := model.NewSkillTag(model.CreateSkillTagInput{SkillId: string(id), Tag: tag})
			if err != nil {
				return err
			}
			if err := tx.Create(mapper.SkillTagToRecord(label)).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ?", id).Take(&skill).Error
	})
	if err != nil {
		return nil, err
	}
	return mapper.RecordToSkill(&skill), nil
}
