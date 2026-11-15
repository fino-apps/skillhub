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

//go:generate mockgen -destination=mocks/skill_version_dao.go -package=mocks . ISkillVersionDao
type ISkillVersionDao interface {
	Get(ctx context.Context, id model.SkillVersionID, opts ...db.Option) (*pmodel.SkillVersion, error)
	List(ctx context.Context, filter model.SkillVersionFilter, afterID model.SkillVersionID, limit int, opts ...db.Option) ([]*pmodel.SkillVersion, error)
	Count(ctx context.Context, filter model.SkillVersionFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.SkillVersion, opts ...db.Option) (model.SkillVersionID, error)
	Update(ctx context.Context, id model.SkillVersionID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.SkillVersionID, opts ...db.Option) error
}

type SkillVersionDaoImpl struct {
	p db.Provider
}

func NewSkillVersionDaoImpl(p db.Provider) ISkillVersionDao {
	return &SkillVersionDaoImpl{
		p: p,
	}
}

func AutoMigrateSkillVersion(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.SkillVersion{}); err != nil {
		return fmt.Errorf("auto migrate skill_version model: %w", err)
	}

	return nil
}

func (m *SkillVersionDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.SkillVersion{})
	return tx
}

func (m *SkillVersionDaoImpl) Get(ctx context.Context, id model.SkillVersionID, opts ...db.Option) (*pmodel.SkillVersion, error) {
	record := &pmodel.SkillVersion{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("skill_version %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *SkillVersionDaoImpl) Create(ctx context.Context, record *pmodel.SkillVersion, opts ...db.Option) (model.SkillVersionID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("skill_version already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *SkillVersionDaoImpl) filteredQuery(ctx context.Context, filter model.SkillVersionFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	return tx
}

func (m *SkillVersionDaoImpl) List(ctx context.Context, filter model.SkillVersionFilter, afterID model.SkillVersionID, limit int, opts ...db.Option) ([]*pmodel.SkillVersion, error) {
	var records []*pmodel.SkillVersion
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

func (m *SkillVersionDaoImpl) Count(ctx context.Context, filter model.SkillVersionFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *SkillVersionDaoImpl) Update(ctx context.Context, id model.SkillVersionID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("skill_version already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill_version %s not found", id)
	}
	return nil
}

func (m *SkillVersionDaoImpl) Delete(ctx context.Context, id model.SkillVersionID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.SkillVersion{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("skill_version %s not found", id)
	}
	return nil
}
