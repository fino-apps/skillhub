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

var _ domainrepo.ISkillRepo = (*SkillRepoImpl)(nil)

type SkillRepoDeps struct {
	SkillDao       dao.ISkillDao
	PageTokenCodec *pagination.CursorCodec
}

type SkillRepoImpl struct {
	skillDao       dao.ISkillDao
	pageTokenCodec *pagination.CursorCodec
}

func NewSkillRepo(deps SkillRepoDeps) domainrepo.ISkillRepo {
	return &SkillRepoImpl{
		skillDao:       deps.SkillDao,
		pageTokenCodec: deps.PageTokenCodec,
	}
}

func (r *SkillRepoImpl) GetSkill(ctx context.Context, id model.SkillID) (*model.Skill, error) {
	if r.skillDao == nil {
		return nil, fmt.Errorf("skill dao is nil")
	}

	record, err := r.skillDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get skill %s: %w", id, err)
	}

	return repomapper.RecordToSkill(record), nil
}

func (r *SkillRepoImpl) ListSkills(ctx context.Context, query model.ListSkillQuery) (model.ListSkillResult, error) {
	if r.skillDao == nil {
		return model.ListSkillResult{}, fmt.Errorf("skill dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill", query.Filter)
	if err != nil {
		return model.ListSkillResult{}, core.NewBadRequestError("resolve skill pagination: %v", err)
	}

	records, err := r.skillDao.List(ctx, query.Filter, model.SkillID(afterID), pageSize+1)
	if err != nil {
		return model.ListSkillResult{}, fmt.Errorf("list skills: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.skillDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListSkillResult{}, fmt.Errorf("count skills: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListSkillResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListSkillResult{
		Items:         repomapper.RecordsToSkills(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *SkillRepoImpl) CreateSkill(ctx context.Context, entity *model.Skill) (*model.Skill, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill is nil")
	}

	if r.skillDao == nil {
		return nil, fmt.Errorf("skill dao is nil")
	}

	record := repomapper.SkillToRecord(entity)
	if _, err := r.skillDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create skill: %w", err)
	}

	return repomapper.RecordToSkill(record), nil
}

func (r *SkillRepoImpl) UpdateSkill(ctx context.Context, entity *model.Skill, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill is nil")
	}

	if r.skillDao == nil {
		return fmt.Errorf("skill dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.skillDao.Update(ctx, snapshot.ID, repomapper.SkillFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update skill %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *SkillRepoImpl) DeleteSkill(ctx context.Context, id model.SkillID) error {
	if r.skillDao == nil {
		return fmt.Errorf("skill dao is nil")
	}

	if err := r.skillDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete skill %s: %w", id, err)
	}

	return nil
}
