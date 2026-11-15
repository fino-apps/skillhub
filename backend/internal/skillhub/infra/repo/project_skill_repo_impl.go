package repo

import (
	"context"
	"fmt"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/pagination"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	domainrepo "github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/dao"
	repomapper "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/mapper"
)

var _ domainrepo.IProjectSkillRepo = (*ProjectSkillRepoImpl)(nil)

type ProjectSkillRepoDeps struct {
	ProjectSkillDao dao.IProjectSkillDao
	PageTokenCodec  *pagination.CursorCodec
}

type ProjectSkillRepoImpl struct {
	projectSkillDao dao.IProjectSkillDao
	pageTokenCodec  *pagination.CursorCodec
}

func NewProjectSkillRepo(deps ProjectSkillRepoDeps) domainrepo.IProjectSkillRepo {
	return &ProjectSkillRepoImpl{
		projectSkillDao: deps.ProjectSkillDao,
		pageTokenCodec:  deps.PageTokenCodec,
	}
}

func (r *ProjectSkillRepoImpl) GetProjectSkill(ctx context.Context, id model.ProjectSkillID) (*model.ProjectSkill, error) {
	if r.projectSkillDao == nil {
		return nil, fmt.Errorf("project_skill dao is nil")
	}

	record, err := r.projectSkillDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project_skill %s: %w", id, err)
	}

	return repomapper.RecordToProjectSkill(record), nil
}

func (r *ProjectSkillRepoImpl) ListProjectSkills(ctx context.Context, query model.ListProjectSkillQuery) (model.ListProjectSkillResult, error) {
	if r.projectSkillDao == nil {
		return model.ListProjectSkillResult{}, fmt.Errorf("project_skill dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "project_skill", query.Filter)
	if err != nil {
		return model.ListProjectSkillResult{}, core.NewBadRequestError("resolve project_skill pagination: %v", err)
	}

	records, err := r.projectSkillDao.List(ctx, query.Filter, model.ProjectSkillID(afterID), pageSize+1)
	if err != nil {
		return model.ListProjectSkillResult{}, fmt.Errorf("list project_skills: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.projectSkillDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListProjectSkillResult{}, fmt.Errorf("count project_skills: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListProjectSkillResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListProjectSkillResult{
		Items:         repomapper.RecordsToProjectSkills(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *ProjectSkillRepoImpl) CreateProjectSkill(ctx context.Context, entity *model.ProjectSkill) (*model.ProjectSkill, error) {
	if entity == nil {
		return nil, fmt.Errorf("project_skill is nil")
	}

	if r.projectSkillDao == nil {
		return nil, fmt.Errorf("project_skill dao is nil")
	}

	record := repomapper.ProjectSkillToRecord(entity)
	if _, err := r.projectSkillDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create project_skill: %w", err)
	}

	return repomapper.RecordToProjectSkill(record), nil
}

func (r *ProjectSkillRepoImpl) UpdateProjectSkill(ctx context.Context, entity *model.ProjectSkill, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("project_skill is nil")
	}

	if r.projectSkillDao == nil {
		return fmt.Errorf("project_skill dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.projectSkillDao.Update(ctx, snapshot.ID, repomapper.ProjectSkillFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update project_skill %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *ProjectSkillRepoImpl) DeleteProjectSkill(ctx context.Context, id model.ProjectSkillID) error {
	if r.projectSkillDao == nil {
		return fmt.Errorf("project_skill dao is nil")
	}

	if err := r.projectSkillDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete project_skill %s: %w", id, err)
	}

	return nil
}
