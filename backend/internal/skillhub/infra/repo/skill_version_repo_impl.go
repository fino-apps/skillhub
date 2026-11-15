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

var _ domainrepo.ISkillVersionRepo = (*SkillVersionRepoImpl)(nil)

type SkillVersionRepoDeps struct {
	SkillVersionDao dao.ISkillVersionDao
	PageTokenCodec  *pagination.CursorCodec
}

type SkillVersionRepoImpl struct {
	skillVersionDao dao.ISkillVersionDao
	pageTokenCodec  *pagination.CursorCodec
}

func NewSkillVersionRepo(deps SkillVersionRepoDeps) domainrepo.ISkillVersionRepo {
	return &SkillVersionRepoImpl{
		skillVersionDao: deps.SkillVersionDao,
		pageTokenCodec:  deps.PageTokenCodec,
	}
}

func (r *SkillVersionRepoImpl) GetSkillVersion(ctx context.Context, id model.SkillVersionID) (*model.SkillVersion, error) {
	if r.skillVersionDao == nil {
		return nil, fmt.Errorf("skill_version dao is nil")
	}

	record, err := r.skillVersionDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get skill_version %s: %w", id, err)
	}

	return repomapper.RecordToSkillVersion(record), nil
}

func (r *SkillVersionRepoImpl) ListSkillVersions(ctx context.Context, query model.ListSkillVersionQuery) (model.ListSkillVersionResult, error) {
	if r.skillVersionDao == nil {
		return model.ListSkillVersionResult{}, fmt.Errorf("skill_version dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill_version", query.Filter)
	if err != nil {
		return model.ListSkillVersionResult{}, core.NewBadRequestError("resolve skill_version pagination: %v", err)
	}

	records, err := r.skillVersionDao.List(ctx, query.Filter, model.SkillVersionID(afterID), pageSize+1)
	if err != nil {
		return model.ListSkillVersionResult{}, fmt.Errorf("list skill_versions: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.skillVersionDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListSkillVersionResult{}, fmt.Errorf("count skill_versions: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListSkillVersionResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListSkillVersionResult{
		Items:         repomapper.RecordsToSkillVersions(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *SkillVersionRepoImpl) CreateSkillVersion(ctx context.Context, entity *model.SkillVersion) (*model.SkillVersion, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill_version is nil")
	}

	if r.skillVersionDao == nil {
		return nil, fmt.Errorf("skill_version dao is nil")
	}

	record := repomapper.SkillVersionToRecord(entity)
	if _, err := r.skillVersionDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create skill_version: %w", err)
	}

	return repomapper.RecordToSkillVersion(record), nil
}

func (r *SkillVersionRepoImpl) UpdateSkillVersion(ctx context.Context, entity *model.SkillVersion, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill_version is nil")
	}

	if r.skillVersionDao == nil {
		return fmt.Errorf("skill_version dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.skillVersionDao.Update(ctx, snapshot.ID, repomapper.SkillVersionFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update skill_version %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *SkillVersionRepoImpl) DeleteSkillVersion(ctx context.Context, id model.SkillVersionID) error {
	if r.skillVersionDao == nil {
		return fmt.Errorf("skill_version dao is nil")
	}

	if err := r.skillVersionDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete skill_version %s: %w", id, err)
	}

	return nil
}
