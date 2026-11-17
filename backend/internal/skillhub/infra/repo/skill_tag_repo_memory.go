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
)

var _ domainrepo.ISkillTagRepo = (*MemorySkillTagRepo)(nil)

type MemorySkillTagRepo struct {
	mu             sync.RWMutex
	items          map[model.SkillTagID]model.SkillTagSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemorySkillTagRepo(codec *pagination.CursorCodec) domainrepo.ISkillTagRepo {
	return &MemorySkillTagRepo{
		items:          make(map[model.SkillTagID]model.SkillTagSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemorySkillTagRepo) GetSkillTag(ctx context.Context, id model.SkillTagID) (*model.SkillTag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("skill_tag %s not found", id)
	}

	return model.RebuildSkillTag(snapshot), nil
}

func (r *MemorySkillTagRepo) ListSkillTags(ctx context.Context, query model.ListSkillTagQuery) (model.ListSkillTagResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.SkillTagID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.SkillTagID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.SkillTagSnapshot, 0, len(r.items))
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
		totalCount = int32(len(snapshots))
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill_tag", query.Filter)
	if err != nil {
		return model.ListSkillTagResult{}, core.NewBadRequestError("resolve skill_tag pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.SkillTagID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListSkillTagResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListSkillTagResult{}, err
		}
	}
	items := make([]*model.SkillTag, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildSkillTag(snapshot))
	}

	return model.ListSkillTagResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemorySkillTagRepo) CreateSkillTag(ctx context.Context, entity *model.SkillTag) (*model.SkillTag, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill_tag is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("skill_tag id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("skill_tag %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildSkillTag(snapshot), nil
}

func (r *MemorySkillTagRepo) UpdateSkillTag(ctx context.Context, entity *model.SkillTag, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill_tag is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("skill_tag id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("skill_tag %s not found", snapshot.ID)
	}

	updated := model.RebuildSkillTag(stored)
	if err := updated.ApplyUpdate(model.UpdateSkillTagInput{
		UpdateMask: updateMask,
		SkillId:    snapshot.SkillId,
		Tag:        snapshot.Tag,
		CreateTime: snapshot.CreateTime,
		UpdateTime: snapshot.UpdateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemorySkillTagRepo) DeleteSkillTag(ctx context.Context, id model.SkillTagID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("skill_tag %s not found", id)
	}
	delete(r.items, id)

	return nil
}
