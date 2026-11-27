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
	"github.com/fino-io/skillhub/backend/internal/skillhub/safecount"
)

var _ domainrepo.ISkillTagRepo = (*SkillTagRepoImpl)(nil)

type SkillTagRepoDeps struct {
	SkillTagDao    dao.ISkillTagDao
	PageTokenCodec *pagination.CursorCodec
}

type SkillTagRepoImpl struct {
	skillTagDao    dao.ISkillTagDao
	pageTokenCodec *pagination.CursorCodec
}

func NewSkillTagRepo(deps SkillTagRepoDeps) domainrepo.ISkillTagRepo {
	return &SkillTagRepoImpl{
		skillTagDao:    deps.SkillTagDao,
		pageTokenCodec: deps.PageTokenCodec,
	}
}

func (r *SkillTagRepoImpl) GetSkillTag(ctx context.Context, id model.SkillTagID) (*model.SkillTag, error) {
	if r.skillTagDao == nil {
		return nil, fmt.Errorf("skill_tag dao is nil")
	}

	record, err := r.skillTagDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get skill_tag %s: %w", id, err)
	}

	return repomapper.RecordToSkillTag(record), nil
}

func (r *SkillTagRepoImpl) ListSkillTags(ctx context.Context, query model.ListSkillTagQuery) (model.ListSkillTagResult, error) {
	if r.skillTagDao == nil {
		return model.ListSkillTagResult{}, fmt.Errorf("skill_tag dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill_tag", query.Filter)
	if err != nil {
		return model.ListSkillTagResult{}, core.NewBadRequestError("resolve skill_tag pagination: %v", err)
	}

	records, err := r.skillTagDao.List(ctx, query.Filter, model.SkillTagID(afterID), pageSize+1)
	if err != nil {
		return model.ListSkillTagResult{}, fmt.Errorf("list skill_tags: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.skillTagDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListSkillTagResult{}, fmt.Errorf("count skill_tags: %w", countErr)
		}
		totalCount = safecount.Int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListSkillTagResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListSkillTagResult{
		Items:         repomapper.RecordsToSkillTags(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *SkillTagRepoImpl) CreateSkillTag(ctx context.Context, entity *model.SkillTag) (*model.SkillTag, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill_tag is nil")
	}

	if r.skillTagDao == nil {
		return nil, fmt.Errorf("skill_tag dao is nil")
	}

	record := repomapper.SkillTagToRecord(entity)
	if _, err := r.skillTagDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create skill_tag: %w", err)
	}

	return repomapper.RecordToSkillTag(record), nil
}

func (r *SkillTagRepoImpl) UpdateSkillTag(ctx context.Context, entity *model.SkillTag, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill_tag is nil")
	}

	if r.skillTagDao == nil {
		return fmt.Errorf("skill_tag dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.skillTagDao.Update(ctx, snapshot.ID, repomapper.SkillTagFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update skill_tag %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *SkillTagRepoImpl) DeleteSkillTag(ctx context.Context, id model.SkillTagID) error {
	if r.skillTagDao == nil {
		return fmt.Errorf("skill_tag dao is nil")
	}

	if err := r.skillTagDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete skill_tag %s: %w", id, err)
	}

	return nil
}
