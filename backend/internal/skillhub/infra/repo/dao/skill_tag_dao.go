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

//go:generate mockgen -destination=mocks/skill_tag_dao.go -package=mocks . ISkillTagDao
type ISkillTagDao interface {
	Get(ctx context.Context, id model.SkillTagID, opts ...db.Option) (*pmodel.SkillTag, error)
	List(ctx context.Context, filter model.SkillTagFilter, afterID model.SkillTagID, limit int, opts ...db.Option) ([]*pmodel.SkillTag, error)
	Count(ctx context.Context, filter model.SkillTagFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.SkillTag, opts ...db.Option) (model.SkillTagID, error)
	Update(ctx context.Context, id model.SkillTagID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.SkillTagID, opts ...db.Option) error
}

type SkillTagDaoImpl struct {
	p db.Provider
}

func NewSkillTagDaoImpl(p db.Provider) ISkillTagDao {
	return &SkillTagDaoImpl{
		p: p,
	}
}

func AutoMigrateSkillTag(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.SkillTag{}); err != nil {
		return fmt.Errorf("auto migrate skill_tag model: %w", err)
	}

	return nil
}

func (m *SkillTagDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.SkillTag{})
	return tx
}

func (m *SkillTagDaoImpl) Get(ctx context.Context, id model.SkillTagID, opts ...db.Option) (*pmodel.SkillTag, error) {
	record := &pmodel.SkillTag{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("skill_tag %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *SkillTagDaoImpl) Create(ctx context.Context, record *pmodel.SkillTag, opts ...db.Option) (model.SkillTagID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("skill_tag already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *SkillTagDaoImpl) filteredQuery(ctx context.Context, filter model.SkillTagFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	return tx
}

func (m *SkillTagDaoImpl) List(ctx context.Context, filter model.SkillTagFilter, afterID model.SkillTagID, limit int, opts ...db.Option) ([]*pmodel.SkillTag, error) {
	var records []*pmodel.SkillTag
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

func (m *SkillTagDaoImpl) Count(ctx context.Context, filter model.SkillTagFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *SkillTagDaoImpl) Update(ctx context.Context, id model.SkillTagID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("skill_tag already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill_tag %s not found", id)
	}
	return nil
}

func (m *SkillTagDaoImpl) Delete(ctx context.Context, id model.SkillTagID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.SkillTag{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill_tag %s not found", id)
	}
	return nil
}
