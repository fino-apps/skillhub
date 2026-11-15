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

//go:generate mockgen -destination=mocks/skill_dao.go -package=mocks . ISkillDao
type ISkillDao interface {
	Get(ctx context.Context, id model.SkillID, opts ...db.Option) (*pmodel.Skill, error)
	List(ctx context.Context, filter model.SkillFilter, afterID model.SkillID, limit int, opts ...db.Option) ([]*pmodel.Skill, error)
	Count(ctx context.Context, filter model.SkillFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.Skill, opts ...db.Option) (model.SkillID, error)
	Update(ctx context.Context, id model.SkillID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.SkillID, opts ...db.Option) error
}

type SkillDaoImpl struct {
	p db.Provider
}

func NewSkillDaoImpl(p db.Provider) ISkillDao {
	return &SkillDaoImpl{
		p: p,
	}
}

func AutoMigrateSkill(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.Skill{}); err != nil {
		return fmt.Errorf("auto migrate skill model: %w", err)
	}

	return nil
}

func (m *SkillDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.Skill{})
	return tx
}

func (m *SkillDaoImpl) Get(ctx context.Context, id model.SkillID, opts ...db.Option) (*pmodel.Skill, error) {
	record := &pmodel.Skill{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("skill %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *SkillDaoImpl) Create(ctx context.Context, record *pmodel.Skill, opts ...db.Option) (model.SkillID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("skill already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *SkillDaoImpl) filteredQuery(ctx context.Context, filter model.SkillFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	if filter.Kind != 0 {
		tx = tx.Where("kind = ?", filter.Kind)
	}
	if filter.Category != "" {
		tx = tx.Where("category = ?", filter.Category)
	}
	return tx
}

func (m *SkillDaoImpl) List(ctx context.Context, filter model.SkillFilter, afterID model.SkillID, limit int, opts ...db.Option) ([]*pmodel.Skill, error) {
	var records []*pmodel.Skill
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

func (m *SkillDaoImpl) Count(ctx context.Context, filter model.SkillFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *SkillDaoImpl) Update(ctx context.Context, id model.SkillID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("skill already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill %s not found", id)
	}
	return nil
}

func (m *SkillDaoImpl) Delete(ctx context.Context, id model.SkillID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.Skill{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill %s not found", id)
	}
	return nil
}
