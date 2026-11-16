package repo

import (
	"context"
	"errors"
	"slices"
	"strings"

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

func NewCatalogRepo(provider db.Provider) *CatalogRepo {
	return &CatalogRepo{provider: provider}
}

func (r *CatalogRepo) AddSkill(ctx context.Context, entry domainrepo.CatalogEntry) (bool, error) {
	var added bool
	err := r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		record := mapper.SkillToRecord(entry.Skill)
		record.DefaultVersionId = string(entry.Version.ID())
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(record)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		added = true
		return tx.Create(mapper.SkillVersionToRecord(entry.Version)).Error
	})
	return added, err
}

func (r *CatalogRepo) AddVersion(ctx context.Context, version *model.SkillVersion, makeDefault bool) (*model.SkillVersion, error) {
	var record *pmodel.SkillVersion
	err := r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		var skill pmodel.Skill
		if err := tx.Where("id = ?", version.SkillId()).Take(&skill).Error; err != nil {
			return translateMissing(err)
		}
		record = mapper.SkillVersionToRecord(version)
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(record)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			query := tx.Where("skill_id = ?", version.SkillId())
			if version.CommitSha() == "" {
				query = query.Where("commit_sha = '' AND content_digest = ?", version.ContentDigest())
			} else {
				query = query.Where("commit_sha = ?", version.CommitSha())
			}
			record = &pmodel.SkillVersion{}
			if err := query.Take(record).Error; err != nil {
				return err
			}
		}
		if result.RowsAffected > 0 && (makeDefault || skill.DefaultVersionId == "") {
			return saveDefault(tx, record)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mapper.RecordToSkillVersion(record), nil
}

func (r *CatalogRepo) SetDefaultVersion(ctx context.Context, version *model.SkillVersion) error {
	return saveDefault(r.provider.NewSession(ctx), mapper.SkillVersionToRecord(version))
}

func saveDefault(tx *gorm.DB, version *pmodel.SkillVersion) error {
	return tx.Model(&pmodel.Skill{}).Where("id = ?", version.SkillId).Updates(map[string]any{
		"default_version_id": version.ID, "name": version.Name,
		"description": version.Description, "update_time": core.Now(),
	}).Error
}

func (r *CatalogRepo) UpdateSnapshot(ctx context.Context, version *model.SkillVersion, digest, description string) error {
	return r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&pmodel.SkillVersion{}).Where("id = ?", version.ID()).Updates(map[string]any{
			"content_digest": digest, "description": description,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&pmodel.Skill{}).Where("id = ? AND default_version_id = ?", version.SkillId(), version.ID()).
			Update("description", description).Error
	})
}

func (r *CatalogRepo) DeleteVersion(ctx context.Context, version *model.SkillVersion) error {
	result := r.provider.NewSession(ctx).Where("id = ? AND NOT EXISTS (SELECT 1 FROM skill WHERE default_version_id = ?)", version.ID(), version.ID()).
		Delete(&pmodel.SkillVersion{})
	if result.Error == nil && result.RowsAffected == 0 {
		return domainrepo.ErrConflict
	}
	return result.Error
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

const displaySource = "COALESCE(NULLIF(upstream_name, ''), import_kind)"

func (r *CatalogRepo) List(ctx context.Context, filter domainrepo.CatalogQuery) ([]*model.Skill, error) {
	query := r.provider.NewSession(ctx).Model(&pmodel.Skill{})
	if filter.Uncategorized {
		query = query.Where("category = ''")
	} else if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Source != "" {
		query = query.Where(displaySource+" = ?", filter.Source)
	}
	if filter.Keyword != "" {
		word := strings.ToLower(filter.Keyword)
		query = query.Where(`instr(lower(name), ?) > 0 OR instr(lower(purpose), ?) > 0 OR
			instr(lower(description), ?) > 0 OR instr(lower(category), ?) > 0 OR
			EXISTS (SELECT 1 FROM skill_tag WHERE skill_tag.skill_id = skill.id AND instr(lower(tag), ?) > 0)`,
			word, word, word, word, word)
	}
	for _, tag := range filter.Tags {
		query = query.Where("EXISTS (SELECT 1 FROM skill_tag WHERE skill_id = skill.id AND tag = ?)", tag)
	}
	var rows []pmodel.Skill
	if err := query.Order("update_time DESC, id DESC").Offset(filter.Offset).Limit(filter.Limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*model.Skill, 0, len(rows))
	for i := range rows {
		result = append(result, mapper.RecordToSkill(&rows[i]))
	}
	return result, nil
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

func (r *CatalogRepo) DefaultVersions(ctx context.Context, skills []*model.Skill) (map[model.SkillID]*model.SkillVersion, error) {
	versions := make(map[model.SkillID]*model.SkillVersion, len(skills))
	if len(skills) == 0 {
		return versions, nil
	}
	ids := make([]string, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.DefaultVersionId())
	}
	var rows []pmodel.SkillVersion
	if err := r.provider.NewSession(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		versions[model.SkillID(rows[i].SkillId)] = mapper.RecordToSkillVersion(&rows[i])
	}
	return versions, nil
}

func (r *CatalogRepo) Facets(ctx context.Context) (categories, sources, tags []string, err error) {
	session := r.provider.NewSession(ctx)
	if err = session.Table("skill").Where("category <> ''").Distinct("category").Order("category").Pluck("category", &categories).Error; err != nil {
		return
	}
	if err = session.Table("skill").Where(displaySource+" <> ''").Distinct(displaySource).Order(displaySource).Pluck(displaySource, &sources).Error; err != nil {
		return
	}
	err = session.Table("skill_tag").Distinct("tag").Order("tag").Pluck("tag", &tags).Error
	return
}

func (r *CatalogRepo) UpdateMetadata(ctx context.Context, changes []domainrepo.Metadata) error {
	return r.provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		for _, change := range changes {
			if err := updateMetadata(tx, change); err != nil {
				return err
			}
		}
		return nil
	})
}

func updateMetadata(tx *gorm.DB, change domainrepo.Metadata) error {
	var skill pmodel.Skill
	if err := tx.Where("id = ?", change.SkillID).Take(&skill).Error; err != nil {
		return translateMissing(err)
	}
	if change.Revision != "" && skill.UpdateTime.String() != change.Revision {
		return domainrepo.ErrConflict
	}
	fields := map[string]any{"update_time": core.Now()}
	for column, value := range map[string]*string{
		"category": change.Category, "purpose": change.Purpose, "notes": change.Notes,
		"upstream_name": change.UpstreamName, "upstream_url": change.UpstreamURL,
	} {
		if value != nil {
			fields[column] = *value
		}
	}
	if err := tx.Model(&skill).Updates(fields).Error; err != nil {
		return err
	}
	if change.ReplaceTags == nil && len(change.AddTags) == 0 && len(change.RemoveTags) == 0 {
		return nil
	}
	query := tx.Where("skill_id = ?", change.SkillID)
	tags := change.ReplaceTags
	if tags == nil {
		if err := query.Model(&pmodel.SkillTag{}).Pluck("tag", &tags).Error; err != nil {
			return err
		}
		tags = slices.DeleteFunc(tags, func(tag string) bool { return slices.Contains(change.RemoveTags, tag) })
		tags = append(tags, change.AddTags...)
	}
	slices.Sort(tags)
	tags = slices.Compact(tags)
	if len(tags) > 100 {
		return errors.New("at most 100 tags per skill")
	}
	if err := tx.Where("skill_id = ?", change.SkillID).Delete(&pmodel.SkillTag{}).Error; err != nil {
		return err
	}
	for _, tag := range tags {
		label, err := model.NewSkillTag(model.CreateSkillTagInput{SkillId: string(change.SkillID), Tag: tag})
		if err != nil {
			return err
		}
		if err := tx.Create(mapper.SkillTagToRecord(label)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *CatalogRepo) StoreTask(ctx context.Context, id string, data []byte) error {
	return r.provider.NewSession(ctx).Exec(
		"INSERT INTO organization_task (id, result) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET result=excluded.result", id, data,
	).Error
}

func (r *CatalogRepo) LoadTask(ctx context.Context, id string) ([]byte, error) {
	var row struct{ Result []byte }
	err := r.provider.NewSession(ctx).Table("organization_task").Where("id = ?", id).Take(&row).Error
	return row.Result, translateMissing(err)
}

func translateMissing(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return core.NewNotFoundError("catalog record not found")
	}
	return err
}

func (r *CatalogRepo) HasVersion(ctx context.Context, skillID, versionID string) (bool, error) {
	var count int64
	err := r.provider.NewSession(ctx).Model(&pmodel.SkillVersion{}).
		Where("id = ? AND skill_id = ?", versionID, skillID).Count(&count).Error
	return count > 0, err
}
