package app

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fino-io/core/go/fino/core"

	"github.com/fino-io/finokit/logs"

	// this service api
	"github.com/fino-io/skillhub/backend/internal/skillhub/app/mapper"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

type ServiceDeps struct {
	SkillhubSvc service.ISkillhubService
	Catalog     repo.CatalogRepo
}

type skillhubServer struct {
	pb.UnimplementedSkillhubServiceServer

	skillhubSvc service.ISkillhubService
	catalog     repo.CatalogRepo
}

// NewService wires the generated transport server to the application service.
func NewService(deps ServiceDeps) pb.SkillhubServiceServer {
	return skillhubServer{
		skillhubSvc: deps.SkillhubSvc,
		catalog:     deps.Catalog,
	}
}

// ListSkills implements Interface. fino:req-fields 3=Category,90=PageSize,91=PageToken
func (s skillhubServer) ListSkills(ctx context.Context, in *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	logs.Ctx(ctx).Debugw("ListSkills")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid list_skills request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.skillhubSvc.ListSkills(ctx, model.ListSkillQuery{
		Filter: model.SkillFilter{
			Category: in.GetCategory(),
		},
		PageSize:       int(in.GetPageSize()),
		PageToken:      in.GetPageToken(),
		SkipTotalCount: true,
	})
	if err != nil {
		if core.IsBadRequestError(err) {
			logs.Ctx(ctx).Warnw("invalid list skills page token", "reason", "invalid_page_token", "error", err)
			return nil, errcode.WrapListSkillhubsBadPageToken(err)
		}
		logs.Ctx(ctx).Errorw("failed to list skills", "error", err)
		return nil, errcode.WrapListSkillhubsFailed(err)
	}

	logs.Ctx(ctx).Debugw("skills listed", "returned_count", len(result.Items), "has_next_page", result.NextPageToken != "")

	ids := make([]model.SkillID, 0, len(result.Items))
	for _, skill := range result.Items {
		ids = append(ids, skill.ID())
	}
	tags, err := s.catalog.Tags(ctx, ids)
	if err != nil {
		return nil, errcode.WrapListSkillhubsFailed(err)
	}
	items := mapper.ToSkillViews(result.Items)
	for i, id := range ids {
		items[i].Tags = tags[id]
	}
	return &pb.ListSkillsResponse{
		Skills:        items,
		NextPageToken: result.NextPageToken,
	}, nil
}

// GetSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) GetSkill(ctx context.Context, in *pb.GetSkillRequest) (*pb.GetSkillResponse, error) {
	logs.Ctx(ctx).Debugw("GetSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid get_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	entity, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId()))
	if err != nil {
		if core.IsNotFoundError(err) {
			logs.Ctx(ctx).Warnw("failed to get skill", "skill_id", in.GetSkillId(), "reason", "not_found", "error", err)
			return nil, errcode.WrapSkillhubNotFound(err)
		}
		logs.Ctx(ctx).Errorw("failed to get skill", "skill_id", in.GetSkillId(), "error", err)
		return nil, errcode.WrapGetSkillhubFailed(err)
	}

	logs.Ctx(ctx).Debugw("skill retrieved", "skill_id", entity.ID())

	tags, err := s.catalog.Tags(ctx, []model.SkillID{entity.ID()})
	if err != nil {
		return nil, errcode.WrapGetSkillhubFailed(err)
	}
	view := mapper.ToSkillView(entity)
	view.Tags = tags[entity.ID()]
	return &pb.GetSkillResponse{Skill: view}, nil
}

// UpdateSkill implements Interface. fino:req-fields 1=SkillId,2=Skill
func (s skillhubServer) UpdateSkill(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	logs.Ctx(ctx).Debugw("UpdateSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid update_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	input := in.GetSkill()
	if input == nil || len(input.GetTags()) > 100 {
		return nil, errcode.NewPlatformBadRequest()
	}
	tags := make([]string, 0, len(input.GetTags()))
	for _, value := range input.GetTags() {
		tag := strings.TrimSpace(value)
		if tag == "" || utf8.RuneCountInString(tag) > 128 {
			return nil, errcode.NewPlatformBadRequest()
		}
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	tags = slices.Compact(tags)
	id := model.SkillID(in.GetSkillId())
	entity, err := s.catalog.UpdateMetadata(ctx, id, strings.TrimSpace(input.GetCategory()), strings.TrimSpace(input.GetNotes()), tags)
	if err != nil {
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapSkillhubNotFound(err)
		}
		return nil, errcode.WrapUpdateSkillhubFailed(err)
	}
	view := mapper.ToSkillView(entity)
	view.Tags = tags
	return &pb.UpdateSkillResponse{Skill: view}, nil
}

// DeleteSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) DeleteSkill(ctx context.Context, in *pb.DeleteSkillRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("DeleteSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid delete_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	id := model.SkillID(in.GetSkillId())

	if err := s.catalog.DeleteSkill(ctx, id); err != nil {
		if core.IsNotFoundError(err) {
			logs.Ctx(ctx).Warnw("failed to delete skill", "skill_id", id, "reason", "not_found", "error", err)
			return nil, errcode.WrapSkillhubNotFound(err)
		}
		logs.Ctx(ctx).Errorw("failed to delete skill", "skill_id", id, "error", err)
		return nil, errcode.WrapDeleteSkillhubsFailed(err)
	}

	logs.Ctx(ctx).Infow("skill deleted", "skill_id", id)

	return &core.Null{}, nil
}

// ListVersions implements Interface. fino:req-fields 1=SkillId,90=PageSize,91=PageToken
func (s skillhubServer) ListVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	logs.Ctx(ctx).Debugw("ListVersions")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid list_versions request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.listVersions(ctx, in)
}

// CreateVersion implements Interface. fino:req-fields 1=SkillId,2=Version
func (s skillhubServer) CreateVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	logs.Ctx(ctx).Debugw("CreateVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid create_version request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.createVersion(ctx, in)
}

// ScanProject implements Interface. fino:req-fields 1=ProjectPath
func (s skillhubServer) ScanProject(ctx context.Context, in *pb.ScanProjectRequest) (*pb.ScanProjectResponse, error) {
	logs.Ctx(ctx).Debugw("ScanProject")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid scan_project request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	skills, err := service.ScanProject(in.GetProjectPath())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}

	items := make([]*pb.SkillCandidate, 0, len(skills))
	for _, skill := range skills {
		items = append(items, &pb.SkillCandidate{
			Path:        skill.Path,
			Name:        skill.Name,
			Description: skill.Description,
		})
	}

	return &pb.ScanProjectResponse{Skills: items}, nil
}

// PreviewImport implements Interface. fino:req-fields 1=Source,2=ScanCollection
func (s skillhubServer) PreviewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	logs.Ctx(ctx).Debugw("PreviewImport")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid preview_import request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.previewImport(ctx, in)
}

// ImportSkill implements Interface. fino:req-fields 1=RepositoryUrl,2=SourceRef,3=CommitSha,4=SkillPaths
func (s skillhubServer) ImportSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	logs.Ctx(ctx).Debugw("ImportSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid import_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.importSkill(ctx, in)
}

// CompareSkills implements Interface. fino:req-fields 1=Left,2=Right
func (s skillhubServer) CompareSkills(ctx context.Context, in *pb.CompareSkillsRequest) (*pb.CompareSkillsResponse, error) {
	logs.Ctx(ctx).Debugw("CompareSkills")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid compare_skills request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if !validSkillLocation(in.GetLeft()) || !validSkillLocation(in.GetRight()) {
		return nil, errcode.NewPlatformBadRequest()
	}

	return s.compareSkills(ctx, in)
}

// GetInstallCommand implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Global
func (s skillhubServer) GetInstallCommand(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
	logs.Ctx(ctx).Debugw("GetInstallCommand")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid get_install_commands request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.getInstallCommand(ctx, in)
}
