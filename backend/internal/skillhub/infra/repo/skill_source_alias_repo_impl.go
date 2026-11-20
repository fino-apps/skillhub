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

var _ domainrepo.ISkillSourceAliasRepo = (*SkillSourceAliasRepoImpl)(nil)

type SkillSourceAliasRepoDeps struct {
	SkillSourceAliasDao dao.ISkillSourceAliasDao
	PageTokenCodec      *pagination.CursorCodec
}

type SkillSourceAliasRepoImpl struct {
	skillSourceAliasDao dao.ISkillSourceAliasDao
	pageTokenCodec      *pagination.CursorCodec
}

func NewSkillSourceAliasRepo(deps SkillSourceAliasRepoDeps) domainrepo.ISkillSourceAliasRepo {
	return &SkillSourceAliasRepoImpl{
		skillSourceAliasDao: deps.SkillSourceAliasDao,
		pageTokenCodec:      deps.PageTokenCodec,
	}
}

func (r *SkillSourceAliasRepoImpl) GetSkillSourceAlias(ctx context.Context, id model.SkillSourceAliasID) (*model.SkillSourceAlias, error) {
	if r.skillSourceAliasDao == nil {
		return nil, fmt.Errorf("skill_source_alias dao is nil")
	}

	record, err := r.skillSourceAliasDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get skill_source_alias %s: %w", id, err)
	}

	return repomapper.RecordToSkillSourceAlias(record), nil
}

func (r *SkillSourceAliasRepoImpl) ListSkillSourceAliass(ctx context.Context, query model.ListSkillSourceAliasQuery) (model.ListSkillSourceAliasResult, error) {
	if r.skillSourceAliasDao == nil {
		return model.ListSkillSourceAliasResult{}, fmt.Errorf("skill_source_alias dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill_source_alias", query.Filter)
	if err != nil {
		return model.ListSkillSourceAliasResult{}, core.NewBadRequestError("resolve skill_source_alias pagination: %v", err)
	}

	records, err := r.skillSourceAliasDao.List(ctx, query.Filter, model.SkillSourceAliasID(afterID), pageSize+1)
	if err != nil {
		return model.ListSkillSourceAliasResult{}, fmt.Errorf("list skill_source_aliass: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.skillSourceAliasDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListSkillSourceAliasResult{}, fmt.Errorf("count skill_source_aliass: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListSkillSourceAliasResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListSkillSourceAliasResult{
		Items:         repomapper.RecordsToSkillSourceAliass(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *SkillSourceAliasRepoImpl) CreateSkillSourceAlias(ctx context.Context, entity *model.SkillSourceAlias) (*model.SkillSourceAlias, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill_source_alias is nil")
	}

	if r.skillSourceAliasDao == nil {
		return nil, fmt.Errorf("skill_source_alias dao is nil")
	}

	record := repomapper.SkillSourceAliasToRecord(entity)
	if _, err := r.skillSourceAliasDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create skill_source_alias: %w", err)
	}

	return repomapper.RecordToSkillSourceAlias(record), nil
}

func (r *SkillSourceAliasRepoImpl) UpdateSkillSourceAlias(ctx context.Context, entity *model.SkillSourceAlias, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill_source_alias is nil")
	}

	if r.skillSourceAliasDao == nil {
		return fmt.Errorf("skill_source_alias dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.skillSourceAliasDao.Update(ctx, snapshot.ID, repomapper.SkillSourceAliasFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update skill_source_alias %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *SkillSourceAliasRepoImpl) DeleteSkillSourceAlias(ctx context.Context, id model.SkillSourceAliasID) error {
	if r.skillSourceAliasDao == nil {
		return fmt.Errorf("skill_source_alias dao is nil")
	}

	if err := r.skillSourceAliasDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete skill_source_alias %s: %w", id, err)
	}

	return nil
}
