package repo

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/pagination"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	domainrepo "github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/safecount"
)

var _ domainrepo.ISkillSourceAliasRepo = (*MemorySkillSourceAliasRepo)(nil)

type MemorySkillSourceAliasRepo struct {
	mu             sync.RWMutex
	items          map[model.SkillSourceAliasID]model.SkillSourceAliasSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemorySkillSourceAliasRepo(codec *pagination.CursorCodec) domainrepo.ISkillSourceAliasRepo {
	return &MemorySkillSourceAliasRepo{
		items:          make(map[model.SkillSourceAliasID]model.SkillSourceAliasSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemorySkillSourceAliasRepo) GetSkillSourceAlias(ctx context.Context, id model.SkillSourceAliasID) (*model.SkillSourceAlias, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("skill_source_alias %s not found", id)
	}

	return model.RebuildSkillSourceAlias(snapshot), nil
}

func (r *MemorySkillSourceAliasRepo) ListSkillSourceAliass(ctx context.Context, query model.ListSkillSourceAliasQuery) (model.ListSkillSourceAliasResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.SkillSourceAliasID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.SkillSourceAliasID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.SkillSourceAliasSnapshot, 0, len(r.items))
	for id, snapshot := range r.items {
		if ids != nil {
			if _, ok := ids[id]; !ok {
				continue
			}
		}
		snapshots = append(snapshots, snapshot)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ID > snapshots[j].ID
	})

	var totalCount int32
	if !query.SkipTotalCount {
		totalCount = safecount.Int32(int64(len(snapshots)))
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill_source_alias", query.Filter)
	if err != nil {
		return model.ListSkillSourceAliasResult{}, core.NewBadRequestError("resolve skill_source_alias pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.SkillSourceAliasID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListSkillSourceAliasResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListSkillSourceAliasResult{}, err
		}
	}
	items := make([]*model.SkillSourceAlias, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildSkillSourceAlias(snapshot))
	}

	return model.ListSkillSourceAliasResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemorySkillSourceAliasRepo) CreateSkillSourceAlias(ctx context.Context, entity *model.SkillSourceAlias) (*model.SkillSourceAlias, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill_source_alias is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("skill_source_alias id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("skill_source_alias %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildSkillSourceAlias(snapshot), nil
}

func (r *MemorySkillSourceAliasRepo) UpdateSkillSourceAlias(ctx context.Context, entity *model.SkillSourceAlias, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill_source_alias is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("skill_source_alias id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("skill_source_alias %s not found", snapshot.ID)
	}

	updated := model.RebuildSkillSourceAlias(stored)
	if err := updated.ApplyUpdate(model.UpdateSkillSourceAliasInput{
		UpdateMask: updateMask,
		SkillId:    snapshot.SkillId,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemorySkillSourceAliasRepo) DeleteSkillSourceAlias(ctx context.Context, id model.SkillSourceAliasID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("skill_source_alias %s not found", id)
	}
	delete(r.items, id)

	return nil
}
