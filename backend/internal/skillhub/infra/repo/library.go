package repo

import (
	"context"
	"errors"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/db"
	"github.com/fino-io/finokit/pagination"
	domain "github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	port "github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/mapper"
	model "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/model/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Library struct {
	provider db.Provider
	cursor   *pagination.CursorCodec
}

type LibraryDeps struct {
	DB             db.Provider
	PageTokenCodec *pagination.CursorCodec
}

func NewLibrary(deps LibraryDeps) *Library {
	return &Library{
		provider: deps.DB,
		cursor:   deps.PageTokenCodec,
	}
}

func (r *Library) session(ctx context.Context) *gorm.DB {
	return r.provider.NewSession(ctx)
}

func (r *Library) Skills(ctx context.Context, filter port.SkillFilter, size int, token string) ([]domain.SkillSnapshot, string, error) {
	query := r.session(ctx).Model(&model.Skill{})

	if filter.Keyword != "" {
		value := filter.Keyword
		query = query.Where("instr(lower(name), lower(?)) > 0 OR instr(lower(description), lower(?)) > 0 OR instr(lower(purpose), lower(?)) > 0 OR id IN (SELECT skill_id FROM skill_tag WHERE instr(lower(tag), lower(?)) > 0)", value, value, value, value)
	}

	if filter.Uncategorized || filter.Category != "" {
		category := filter.Category
		if filter.Uncategorized {
			category = ""
		}
		query = query.Where("category = ?", category)
	}

	if filter.Source != "" {
		query = query.Where("import_kind = ? OR upstream_name = ?", filter.Source, filter.Source)
	}

	for _, tag := range filter.Tags {
		query = query.Where("id IN (SELECT skill_id FROM skill_tag WHERE tag = ?)", tag)
	}

	rows, next, err := page[model.Skill](r.cursor, query, "skills", filter, size, token, func(x model.Skill) string { return string(x.ID) })

	result := make([]domain.SkillSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.SkillSnapshot(row))
	}
	return result, next, err
}

func (r *Library) Versions(ctx context.Context, skill string, size int, token string) ([]domain.SkillVersionSnapshot, string, error) {
	query := r.session(ctx).Where("skill_id = ?", skill)
	rows, next, err := page[model.SkillVersion](r.cursor, query, "versions", skill, size, token, func(x model.SkillVersion) string { return string(x.ID) })
	return versionSnapshots(rows), next, err
}

func page[T any](codec *pagination.CursorCodec, query *gorm.DB, scope string, filters any, size int, token string, id func(T) string) ([]T, string, error) {
	after, size, binding, err := pagination.ResolveKeyset(codec, token, size, scope, filters)
	if err != nil {
		return nil, "", port.ErrPageToken
	}

	if after != "" {
		query = query.Where("id < ?", after)
	}

	var rows []T
	if err := query.Order("id DESC").Limit(size + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	if len(rows) <= size {
		return rows, "", nil
	}

	next, err := codec.EncodeKeyset(id(rows[size-1]), binding)
	return rows[:size], next, err
}

var _ port.Library = (*Library)(nil)

type libraryTx struct{ db *gorm.DB }

func (r *Library) Transaction(ctx context.Context, run func(port.LibraryTx) error) error {
	return r.session(ctx).Transaction(func(tx *gorm.DB) error { return run(libraryTx{tx}) })
}

func (r *Library) Skill(ctx context.Context, id string) (domain.SkillSnapshot, error) {
	return (libraryTx{r.session(ctx)}).Skill(id)
}

func (r *Library) Version(ctx context.Context, skill, id string) (domain.SkillVersionSnapshot, error) {
	return (libraryTx{r.session(ctx)}).Version(skill, id)
}

func (r *Library) Tags(ctx context.Context, ids []string) (map[string][]string, error) {
	var tags []model.SkillTag
	err := r.session(ctx).Where("skill_id IN ?", ids).Order("tag").Find(&tags).Error

	result := make(map[string][]string)
	for _, tag := range tags {
		result[tag.SkillId] = append(result[tag.SkillId], tag.Tag)
	}
	return result, err
}

func (r *Library) DefaultVersions(ctx context.Context, ids []string) ([]domain.SkillVersionSnapshot, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var rows []model.SkillVersion
	err := r.session(ctx).Where("id IN ?", ids).Find(&rows).Error
	return versionSnapshots(rows), err
}

func versionSnapshots(rows []model.SkillVersion) []domain.SkillVersionSnapshot {
	result := make([]domain.SkillVersionSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.SkillVersionSnapshot(row))
	}
	return result
}

func (r *Library) Facets(ctx context.Context) (port.Facets, error) {
	db := r.session(ctx)
	var result port.Facets

	for _, field := range []struct {
		model  any
		column string
		output *[]string
	}{
		{&model.Skill{}, "category", &result.Categories},
		{&model.Skill{}, "import_kind", &result.Sources},
		{&model.SkillTag{}, "tag", &result.Tags},
	} {
		if err := db.Model(field.model).Where(field.column+" <> ''").Distinct().Order(field.column).Pluck(field.column, field.output).Error; err != nil {
			return result, err
		}
	}
	return result, nil
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return port.ErrNotFound
	}
	return err
}

func (tx libraryTx) Skill(id string) (domain.SkillSnapshot, error) {
	var row model.Skill
	err := tx.db.Take(&row, "id = ?", id).Error
	return domain.SkillSnapshot(row), notFound(err)
}

func (tx libraryTx) SkillBySource(key string) (domain.SkillSnapshot, error) {
	var row model.Skill
	err := tx.db.Take(&row, "source_key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && key != "" {
		aliases := tx.db.Model(&model.SkillSourceAlias{}).Select("skill_id").Where("source_key = ?", key)
		err = tx.db.Take(&row, "id IN (?)", aliases).Error
	}
	return domain.SkillSnapshot(row), notFound(err)
}

func (tx libraryTx) Version(skill, id string) (domain.SkillVersionSnapshot, error) {
	db := tx.db
	if skill != "" {
		db = db.Where("skill_id = ?", skill)
	}

	var row model.SkillVersion
	err := db.Take(&row, "id = ?", id).Error
	return domain.SkillVersionSnapshot(row), notFound(err)
}

func (tx libraryTx) MatchingVersion(skill, commit, digest string) (domain.SkillVersionSnapshot, error) {
	db := tx.db.Where("skill_id = ?", skill)
	if commit == "" {
		db = db.Where("commit_sha = '' AND content_digest = ?", digest)
	} else {
		db = db.Where("commit_sha = ? AND (content_digest = ? OR content_digest = '')", commit, digest)
	}

	var row model.SkillVersion
	err := db.Take(&row).Error
	return domain.SkillVersionSnapshot(row), notFound(err)
}

func (tx libraryTx) SkillVersions(skill string) ([]domain.SkillVersionSnapshot, error) {
	var rows []model.SkillVersion
	err := tx.db.Where("skill_id = ?", skill).Order("id").Find(&rows).Error
	return versionSnapshots(rows), err
}

func (tx libraryTx) MoveVersion(version domain.SkillVersionSnapshot) error {
	return tx.db.Model(&model.SkillVersion{}).Where("id = ?", version.ID).Updates(map[string]any{
		"skill_id": version.SkillId, "upstream_commit": version.UpstreamCommit,
		"upstream_repository": version.UpstreamRepository, "upstream_path": version.UpstreamPath,
	}).Error
}

func (tx libraryTx) CreateSourceAlias(alias port.SourceAlias) error {
	return tx.db.Create(&model.SkillSourceAlias{
		ID: domain.SkillSourceAliasID(uuid.NewString()), SkillId: alias.SkillID, SourceKey: alias.SourceKey,
	}).Error
}

func (tx libraryTx) MoveSourceAliases(source, target string) error {
	return tx.db.Model(&model.SkillSourceAlias{}).Where("skill_id = ?", source).Update("skill_id", target).Error
}

func (tx libraryTx) BackupData() (port.LibraryData, error) {
	var data port.LibraryData
	var skills []model.Skill
	var tags []model.SkillTag
	var versions []model.SkillVersion
	var aliases []model.SkillSourceAlias
	if err := tx.db.Order("id").Find(&skills).Error; err != nil {
		return data, err
	}
	if err := tx.db.Order("id").Find(&tags).Error; err != nil {
		return data, err
	}
	if err := tx.db.Order("id").Find(&versions).Error; err != nil {
		return data, err
	}
	if err := tx.db.Order("source_key").Find(&aliases).Error; err != nil {
		return data, err
	}
	for _, skill := range skills {
		data.Skills = append(data.Skills, domain.SkillSnapshot(skill))
	}
	for _, tag := range tags {
		data.Tags = append(data.Tags, domain.SkillTagSnapshot(tag))
	}
	data.Versions = versionSnapshots(versions)
	for _, alias := range aliases {
		data.Aliases = append(data.Aliases, port.SourceAlias{SkillID: alias.SkillId, SourceKey: alias.SourceKey})
	}
	return data, nil
}

func (tx libraryTx) CreateTag(tag domain.SkillTagSnapshot) error {
	row := model.SkillTag(tag)
	return tx.db.Create(&row).Error
}

func (tx libraryTx) CreateSkill(skill domain.SkillSnapshot) error {
	row := model.Skill(skill)
	return tx.db.Create(&row).Error
}

func (tx libraryTx) CreateVersion(version domain.SkillVersionSnapshot) error {
	row := model.SkillVersion(version)
	return tx.db.Create(&row).Error
}

func (tx libraryTx) UpdateSkill(skill domain.SkillSnapshot, fields ...string) error {
	return tx.db.Model(&model.Skill{}).Where("id = ?", skill.ID).Updates(mapper.SkillFields(skill, fields)).Error
}

func (tx libraryTx) ChangeTags(skill string, clear bool, add, remove []string) error {
	query := tx.db.Where("skill_id = ?", skill)
	switch {
	case clear:
		if err := query.Delete(&model.SkillTag{}).Error; err != nil {
			return err
		}
	case len(remove) > 0:
		if err := query.Where("tag IN ?", remove).Delete(&model.SkillTag{}).Error; err != nil {
			return err
		}
	}

	for _, tag := range add {
		now := core.Now()
		row := model.SkillTag{ID: domain.SkillTagID(uuid.NewString()), SkillId: skill, Tag: tag, CreateTime: now, UpdateTime: now}
		if err := tx.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (tx libraryTx) DeleteVersion(version domain.SkillVersionSnapshot) error {
	row := model.SkillVersion(version)
	return tx.db.Delete(&row).Error
}

func (tx libraryTx) DeleteSkill(skill domain.SkillSnapshot) ([]string, error) {
	id := string(skill.ID)
	var versions []string
	if err := tx.db.Model(&model.SkillVersion{}).Where("skill_id = ?", id).Pluck("id", &versions).Error; err != nil {
		return nil, err
	}

	if err := tx.db.Where("skill_id = ?", id).Delete(&model.SkillVersion{}).Error; err != nil {
		return nil, err
	}
	if err := tx.db.Where("skill_id = ?", id).Delete(&model.SkillTag{}).Error; err != nil {
		return nil, err
	}

	if err := tx.db.Where("skill_id = ?", id).Delete(&model.SkillSourceAlias{}).Error; err != nil {
		return nil, err
	}

	row := model.Skill(skill)
	return versions, tx.db.Delete(&row).Error
}
