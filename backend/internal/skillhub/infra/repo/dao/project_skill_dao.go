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

//go:generate mockgen -destination=mocks/project_skill_dao.go -package=mocks . IProjectSkillDao
type IProjectSkillDao interface {
	Get(ctx context.Context, id model.ProjectSkillID, opts ...db.Option) (*pmodel.ProjectSkill, error)
	List(ctx context.Context, filter model.ProjectSkillFilter, afterID model.ProjectSkillID, limit int, opts ...db.Option) ([]*pmodel.ProjectSkill, error)
	Count(ctx context.Context, filter model.ProjectSkillFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.ProjectSkill, opts ...db.Option) (model.ProjectSkillID, error)
	Update(ctx context.Context, id model.ProjectSkillID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.ProjectSkillID, opts ...db.Option) error
}

type ProjectSkillDaoImpl struct {
	p db.Provider
}

func NewProjectSkillDaoImpl(p db.Provider) IProjectSkillDao {
	return &ProjectSkillDaoImpl{
		p: p,
	}
}

func AutoMigrateProjectSkill(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.ProjectSkill{}); err != nil {
		return fmt.Errorf("auto migrate project_skill model: %w", err)
	}

	return nil
}

func (m *ProjectSkillDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.ProjectSkill{})
	return tx
}

func (m *ProjectSkillDaoImpl) Get(ctx context.Context, id model.ProjectSkillID, opts ...db.Option) (*pmodel.ProjectSkill, error) {
	record := &pmodel.ProjectSkill{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("project_skill %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *ProjectSkillDaoImpl) Create(ctx context.Context, record *pmodel.ProjectSkill, opts ...db.Option) (model.ProjectSkillID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("project_skill already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *ProjectSkillDaoImpl) filteredQuery(ctx context.Context, filter model.ProjectSkillFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	return tx
}

func (m *ProjectSkillDaoImpl) List(ctx context.Context, filter model.ProjectSkillFilter, afterID model.ProjectSkillID, limit int, opts ...db.Option) ([]*pmodel.ProjectSkill, error) {
	var records []*pmodel.ProjectSkill
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

func (m *ProjectSkillDaoImpl) Count(ctx context.Context, filter model.ProjectSkillFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *ProjectSkillDaoImpl) Update(ctx context.Context, id model.ProjectSkillID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("project_skill already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("project_skill %s not found", id)
	}
	return nil
}

func (m *ProjectSkillDaoImpl) Delete(ctx context.Context, id model.ProjectSkillID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.ProjectSkill{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("project_skill %s not found", id)
	}
	return nil
}
