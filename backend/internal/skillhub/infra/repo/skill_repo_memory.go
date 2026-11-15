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

var _ domainrepo.ISkillRepo = (*MemorySkillRepo)(nil)

type MemorySkillRepo struct {
	mu             sync.RWMutex
	items          map[model.SkillID]model.SkillSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemorySkillRepo(codec *pagination.CursorCodec) domainrepo.ISkillRepo {
	return &MemorySkillRepo{
		items:          make(map[model.SkillID]model.SkillSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemorySkillRepo) GetSkill(ctx context.Context, id model.SkillID) (*model.Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("skill %s not found", id)
	}

	return model.RebuildSkill(snapshot), nil
}

func (r *MemorySkillRepo) ListSkills(ctx context.Context, query model.ListSkillQuery) (model.ListSkillResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.SkillID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.SkillID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.SkillSnapshot, 0, len(r.items))
	for id, snapshot := range r.items {
		if ids != nil {
			if _, ok := ids[id]; !ok {
				continue
			}
		}
		if query.Filter.Category != "" && snapshot.Category != query.Filter.Category {
			continue
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

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill", query.Filter)
	if err != nil {
		return model.ListSkillResult{}, core.NewBadRequestError("resolve skill pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.SkillID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListSkillResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListSkillResult{}, err
		}
	}
	items := make([]*model.Skill, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildSkill(snapshot))
	}

	return model.ListSkillResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemorySkillRepo) CreateSkill(ctx context.Context, entity *model.Skill) (*model.Skill, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("skill id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("skill %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildSkill(snapshot), nil
}

func (r *MemorySkillRepo) UpdateSkill(ctx context.Context, entity *model.Skill, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("skill id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("skill %s not found", snapshot.ID)
	}

	updated := model.RebuildSkill(stored)
	if err := updated.ApplyUpdate(model.UpdateSkillInput{
		UpdateMask:     updateMask,
		Name:           snapshot.Name,
		Description:    snapshot.Description,
		Category:       snapshot.Category,
		Notes:          snapshot.Notes,
		RepositoryUrl:  snapshot.RepositoryUrl,
		RepositoryPath: snapshot.RepositoryPath,
		LocalPath:      snapshot.LocalPath,
		CreateTime:     snapshot.CreateTime,
		UpdateTime:     snapshot.UpdateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemorySkillRepo) DeleteSkill(ctx context.Context, id model.SkillID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("skill %s not found", id)
	}
	delete(r.items, id)

	return nil
}
