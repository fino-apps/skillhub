package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/db"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	pmodel "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/model/sqlite"
	"gorm.io/gorm"
)

//go:generate mockgen -destination=mocks/skill_source_alias_dao.go -package=mocks . ISkillSourceAliasDao
type ISkillSourceAliasDao interface {
	Get(ctx context.Context, id model.SkillSourceAliasID, opts ...db.Option) (*pmodel.SkillSourceAlias, error)
	List(ctx context.Context, filter model.SkillSourceAliasFilter, afterID model.SkillSourceAliasID, limit int, opts ...db.Option) ([]*pmodel.SkillSourceAlias, error)
	Count(ctx context.Context, filter model.SkillSourceAliasFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.SkillSourceAlias, opts ...db.Option) (model.SkillSourceAliasID, error)
	Update(ctx context.Context, id model.SkillSourceAliasID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.SkillSourceAliasID, opts ...db.Option) error
}

type SkillSourceAliasDaoImpl struct {
	p db.Provider
}

func NewSkillSourceAliasDaoImpl(p db.Provider) ISkillSourceAliasDao {
	return &SkillSourceAliasDaoImpl{
		p: p,
	}
}

func AutoMigrateSkillSourceAlias(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.SkillSourceAlias{}); err != nil {
		return fmt.Errorf("auto migrate skill_source_alias model: %w", err)
	}

	return nil
}

func (m *SkillSourceAliasDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.SkillSourceAlias{})
	return tx
}

func (m *SkillSourceAliasDaoImpl) Get(ctx context.Context, id model.SkillSourceAliasID, opts ...db.Option) (*pmodel.SkillSourceAlias, error) {
	record := &pmodel.SkillSourceAlias{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("skill_source_alias %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *SkillSourceAliasDaoImpl) Create(ctx context.Context, record *pmodel.SkillSourceAlias, opts ...db.Option) (model.SkillSourceAliasID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("skill_source_alias already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *SkillSourceAliasDaoImpl) filteredQuery(ctx context.Context, filter model.SkillSourceAliasFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	return tx
}

func (m *SkillSourceAliasDaoImpl) List(ctx context.Context, filter model.SkillSourceAliasFilter, afterID model.SkillSourceAliasID, limit int, opts ...db.Option) ([]*pmodel.SkillSourceAlias, error) {
	var records []*pmodel.SkillSourceAlias
	tx := m.filteredQuery(ctx, filter, opts...)
	tx = tx.Order("id DESC")
	if afterID != "" {
		tx = tx.Where("id < ?", afterID)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (m *SkillSourceAliasDaoImpl) Count(ctx context.Context, filter model.SkillSourceAliasFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *SkillSourceAliasDaoImpl) Update(ctx context.Context, id model.SkillSourceAliasID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("skill_source_alias already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill_source_alias %s not found", id)
	}
	return nil
}

func (m *SkillSourceAliasDaoImpl) Delete(ctx context.Context, id model.SkillSourceAliasID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.SkillSourceAlias{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill_source_alias %s not found", id)
	}
	return nil
}
