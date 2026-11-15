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

var _ domainrepo.IProjectSkillRepo = (*MemoryProjectSkillRepo)(nil)

type MemoryProjectSkillRepo struct {
	mu             sync.RWMutex
	items          map[model.ProjectSkillID]model.ProjectSkillSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemoryProjectSkillRepo(codec *pagination.CursorCodec) domainrepo.IProjectSkillRepo {
	return &MemoryProjectSkillRepo{
		items:          make(map[model.ProjectSkillID]model.ProjectSkillSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemoryProjectSkillRepo) GetProjectSkill(ctx context.Context, id model.ProjectSkillID) (*model.ProjectSkill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("project_skill %s not found", id)
	}

	return model.RebuildProjectSkill(snapshot), nil
}

func (r *MemoryProjectSkillRepo) ListProjectSkills(ctx context.Context, query model.ListProjectSkillQuery) (model.ListProjectSkillResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.ProjectSkillID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.ProjectSkillID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.ProjectSkillSnapshot, 0, len(r.items))
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

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "project_skill", query.Filter)
	if err != nil {
		return model.ListProjectSkillResult{}, core.NewBadRequestError("resolve project_skill pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.ProjectSkillID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListProjectSkillResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListProjectSkillResult{}, err
		}
	}
	items := make([]*model.ProjectSkill, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildProjectSkill(snapshot))
	}

	return model.ListProjectSkillResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemoryProjectSkillRepo) CreateProjectSkill(ctx context.Context, entity *model.ProjectSkill) (*model.ProjectSkill, error) {
	if entity == nil {
		return nil, fmt.Errorf("project_skill is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("project_skill id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("project_skill %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildProjectSkill(snapshot), nil
}

func (r *MemoryProjectSkillRepo) UpdateProjectSkill(ctx context.Context, entity *model.ProjectSkill, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("project_skill is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("project_skill id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("project_skill %s not found", snapshot.ID)
	}

	updated := model.RebuildProjectSkill(stored)
	if err := updated.ApplyUpdate(model.UpdateProjectSkillInput{
		UpdateMask:    updateMask,
		ProjectPath:   snapshot.ProjectPath,
		SkillPath:     snapshot.SkillPath,
		SkillId:       snapshot.SkillId,
		BaseVersionId: snapshot.BaseVersionId,
		BaseDigest:    snapshot.BaseDigest,
		CreateTime:    snapshot.CreateTime,
		UpdateTime:    snapshot.UpdateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemoryProjectSkillRepo) DeleteProjectSkill(ctx context.Context, id model.ProjectSkillID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("project_skill %s not found", id)
	}
	delete(r.items, id)

	return nil
}
