package app

import (
	"context"

	"github.com/fino-io/core/go/fino/core"

	"github.com/fino-io/finokit/logs"

	// this service api
	"github.com/fino-io/skillhub/backend/internal/skillhub/app/mapper"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

type ServiceDeps struct {
	SkillhubSvc service.ISkillhubService
	Intake      *service.Intake
}

type skillhubServer struct {
	pb.UnimplementedSkillhubServiceServer

	skillhubSvc service.ISkillhubService
	intake      *service.Intake
}

// NewService wires the generated transport server to the application service.
func NewService(deps ServiceDeps) pb.SkillhubServiceServer {
	return skillhubServer{
		skillhubSvc: deps.SkillhubSvc,
		intake:      deps.Intake,
	}
}

// ListSkills implements Interface. fino:req-fields 1=Keyword,2=Kind,3=Category,4=Tag,90=PageSize,91=PageToken
func (s skillhubServer) ListSkills(ctx context.Context, in *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	logs.Ctx(ctx).Debugw("ListSkills")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid list_skills request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.skillhubSvc.ListSkills(ctx, model.ListSkillQuery{
		Filter: model.SkillFilter{
			Kind:     int32(in.GetKind()),
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

	return &pb.ListSkillsResponse{
		Skills:        mapper.ToSkillViews(result.Items),
		NextPageToken: result.NextPageToken,
	}, nil
}

// ListSkillFilters implements Interface. fino:req-fields
func (s skillhubServer) ListSkillFilters(ctx context.Context, in *pb.ListSkillFiltersRequest) (*pb.ListSkillFiltersResponse, error) {
	logs.Ctx(ctx).Debugw("ListSkillFilters")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid list_skill_filters request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.ListSkillFilters(ctx, in)
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

	return &pb.GetSkillResponse{
		Skill: mapper.ToSkillView(entity),
	}, nil
}

// CreateSkill implements Interface. fino:req-fields 1=Skill
func (s skillhubServer) CreateSkill(ctx context.Context, in *pb.CreateSkillRequest) (*pb.CreateSkillResponse, error) {
	logs.Ctx(ctx).Debugw("CreateSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid create_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	input, err := mapper.ToCreateSkillInput(in.GetSkill())
	if err != nil {
		logs.Ctx(ctx).Warnw("invalid create_skill_input", "error", err)
		return nil, errcode.WrapCreateSkillhubInputInvalid(err)
	}

	entity, err := s.skillhubSvc.CreateSkill(ctx, input)
	if err != nil {
		if core.IsAlreadyExistsError(err) {
			logs.Ctx(ctx).Warnw("failed to create skill", "reason", "already_exists", "error", err)
			return nil, errcode.WrapPlatformConflict(err)
		}
		logs.Ctx(ctx).Errorw("failed to create skill", "error", err)
		return nil, errcode.WrapCreateSkillhubFailed(err)
	}

	logs.Ctx(ctx).Infow("skill created", "skill_id", entity.ID())

	return &pb.CreateSkillResponse{
		Skill: mapper.ToSkillView(entity),
	}, nil
}

// UpdateSkill implements Interface. fino:req-fields 1=SkillId,2=Skill,80=UpdateMask
func (s skillhubServer) UpdateSkill(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	logs.Ctx(ctx).Debugw("UpdateSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid update_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	input, err := mapper.ToUpdateSkillInput(in.GetSkill(), in.GetUpdateMask().GetPaths())
	if err != nil {
		logs.Ctx(ctx).Warnw("invalid update_skill_input", "error", err)
		return nil, errcode.WrapUpdateSkillhubInputInvalid(err)
	}

	id := model.SkillID(in.GetSkillId())
	entity, err := s.skillhubSvc.UpdateSkill(ctx, id, input)
	if err != nil {
		if core.IsNotFoundError(err) {
			logs.Ctx(ctx).Warnw("failed to update skill", "skill_id", id, "reason", "not_found", "error", err)
			return nil, errcode.WrapSkillhubNotFound(err)
		}
		if core.IsAlreadyExistsError(err) {
			logs.Ctx(ctx).Warnw("failed to update skill", "skill_id", id, "reason", "already_exists", "error", err)
			return nil, errcode.WrapPlatformConflict(err)
		}
		logs.Ctx(ctx).Errorw("failed to update skill", "skill_id", id, "error", err)
		return nil, errcode.WrapUpdateSkillhubFailed(err)
	}

	logs.Ctx(ctx).Infow("skill updated", "skill_id", entity.ID(), "updated_fields", input.UpdateMask)

	return &pb.UpdateSkillResponse{
		Skill: mapper.ToSkillView(entity),
	}, nil
}

// DeleteSkill implements Interface. fino:req-fields 1=SkillId,2=DeleteContent
func (s skillhubServer) DeleteSkill(ctx context.Context, in *pb.DeleteSkillRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("DeleteSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid delete_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	id := model.SkillID(in.GetSkillId())

	if err := s.skillhubSvc.DeleteSkill(ctx, id); err != nil {
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

// CreateVersion implements Interface. fino:req-fields 1=SkillId,2=Version,3=ExpectedDigest
func (s skillhubServer) CreateVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	logs.Ctx(ctx).Debugw("CreateVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid create_version request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.CreateVersion(ctx, in)
}

// RestoreVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=ExpectedDigest
func (s skillhubServer) RestoreVersion(ctx context.Context, in *pb.RestoreVersionRequest) (*pb.RestoreVersionResponse, error) {
	logs.Ctx(ctx).Debugw("RestoreVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid restore_version request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.RestoreVersion(ctx, in)
}

// ListFiles implements Interface. fino:req-fields 1=Location
func (s skillhubServer) ListFiles(ctx context.Context, in *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	logs.Ctx(ctx).Debugw("ListFiles")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid list_files request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if !validSkillLocation(in.GetLocation()) {
		return nil, errcode.NewPlatformBadRequest()
	}

	return s.UnimplementedSkillhubServiceServer.ListFiles(ctx, in)
}

// GetFile implements Interface. fino:req-fields 1=Location,2=Path
func (s skillhubServer) GetFile(ctx context.Context, in *pb.GetFileRequest) (*pb.GetFileResponse, error) {
	logs.Ctx(ctx).Debugw("GetFile")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid get_file request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if !validSkillLocation(in.GetLocation()) {
		return nil, errcode.NewPlatformBadRequest()
	}

	return s.UnimplementedSkillhubServiceServer.GetFile(ctx, in)
}

// PutFile implements Interface. fino:req-fields 1=SkillId,2=Path,3=Text,4=ExpectedDigest
func (s skillhubServer) PutFile(ctx context.Context, in *pb.PutFileRequest) (*pb.PutFileResponse, error) {
	logs.Ctx(ctx).Debugw("PutFile")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid put_file request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.PutFile(ctx, in)
}

// DeleteFile implements Interface. fino:req-fields 1=SkillId,2=Path,3=ExpectedDigest
func (s skillhubServer) DeleteFile(ctx context.Context, in *pb.DeleteFileRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("DeleteFile")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid delete_file request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.DeleteFile(ctx, in)
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

	items := make([]*pb.ProjectSkillView, 0, len(skills))
	for _, skill := range skills {
		items = append(items, &pb.ProjectSkillView{
			Path:        skill.Path,
			Name:        skill.Name,
			Description: skill.Description,
			Status:      pb.ProjectSkillStatus_PROJECT_SKILL_STATUS_NO_BASELINE,
		})
	}

	return &pb.ScanProjectResponse{Skills: items}, nil
}

// PreviewImport implements Interface. fino:req-fields 1=Source,2=TargetSkillId,3=ProjectPath
func (s skillhubServer) PreviewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	logs.Ctx(ctx).Debugw("PreviewImport")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid preview_import request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if !validSkillLocation(in.GetSource()) {
		return nil, errcode.NewPlatformBadRequest()
	}

	return s.previewImport(ctx, in)
}

// ImportSkill implements Interface. fino:req-fields 1=Source,2=TargetSkillId,3=ProjectPath,4=ExpectedSourceDigest,5=ExpectedTargetDigest,6=IncludedFiles,7=Version
func (s skillhubServer) ImportSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	logs.Ctx(ctx).Debugw("ImportSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid import_skill request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if !validSkillLocation(in.GetSource()) {
		return nil, errcode.NewPlatformBadRequest()
	}

	return s.importSkill(ctx, in)
}

// ResolveSource implements Interface. fino:req-fields 1=Source
func (s skillhubServer) ResolveSource(ctx context.Context, in *pb.ResolveSourceRequest) (*pb.ResolveSourceResponse, error) {
	logs.Ctx(ctx).Debugw("ResolveSource")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid resolve_source request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.ResolveSource(ctx, in)
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

	return s.UnimplementedSkillhubServiceServer.CompareSkills(ctx, in)
}

// GetInstallCommands implements Interface. fino:req-fields 1=Skills,2=ProjectPath,3=Global
func (s skillhubServer) GetInstallCommands(ctx context.Context, in *pb.GetInstallCommandsRequest) (*pb.GetInstallCommandsResponse, error) {
	logs.Ctx(ctx).Debugw("GetInstallCommands")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid get_install_commands request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	return s.UnimplementedSkillhubServiceServer.GetInstallCommands(ctx, in)
}
