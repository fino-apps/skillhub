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

var _ domainrepo.ISkillVersionRepo = (*MemorySkillVersionRepo)(nil)

type MemorySkillVersionRepo struct {
	mu             sync.RWMutex
	items          map[model.SkillVersionID]model.SkillVersionSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemorySkillVersionRepo(codec *pagination.CursorCodec) domainrepo.ISkillVersionRepo {
	return &MemorySkillVersionRepo{
		items:          make(map[model.SkillVersionID]model.SkillVersionSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemorySkillVersionRepo) GetSkillVersion(ctx context.Context, id model.SkillVersionID) (*model.SkillVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("skill_version %s not found", id)
	}

	return model.RebuildSkillVersion(snapshot), nil
}

func (r *MemorySkillVersionRepo) ListSkillVersions(ctx context.Context, query model.ListSkillVersionQuery) (model.ListSkillVersionResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.SkillVersionID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.SkillVersionID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.SkillVersionSnapshot, 0, len(r.items))
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

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "skill_version", query.Filter)
	if err != nil {
		return model.ListSkillVersionResult{}, core.NewBadRequestError("resolve skill_version pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.SkillVersionID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListSkillVersionResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListSkillVersionResult{}, err
		}
	}
	items := make([]*model.SkillVersion, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildSkillVersion(snapshot))
	}

	return model.ListSkillVersionResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemorySkillVersionRepo) CreateSkillVersion(ctx context.Context, entity *model.SkillVersion) (*model.SkillVersion, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill_version is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("skill_version id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("skill_version %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildSkillVersion(snapshot), nil
}

func (r *MemorySkillVersionRepo) UpdateSkillVersion(ctx context.Context, entity *model.SkillVersion, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("skill_version is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("skill_version id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("skill_version %s not found", snapshot.ID)
	}

	updated := model.RebuildSkillVersion(stored)
	if err := updated.ApplyUpdate(model.UpdateSkillVersionInput{
		UpdateMask:    updateMask,
		SkillId:       snapshot.SkillId,
		Name:          snapshot.Name,
		Notes:         snapshot.Notes,
		SourceRef:     snapshot.SourceRef,
		CommitSha:     snapshot.CommitSha,
		ContentDigest: snapshot.ContentDigest,
		CreateTime:    snapshot.CreateTime,
		UpdateTime:    snapshot.UpdateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemorySkillVersionRepo) DeleteSkillVersion(ctx context.Context, id model.SkillVersionID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("skill_version %s not found", id)
	}
	delete(r.items, id)

	return nil
}
