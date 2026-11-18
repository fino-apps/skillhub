package app

import (
	"context"
	"strings"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

type ServiceDeps struct {
	Library *service.Library
}

type skillhubServer struct {
	pb.UnimplementedSkillhubServiceServer

	library *service.Library
}

func NewService(deps ServiceDeps) pb.SkillhubServiceServer {
	return skillhubServer{
		library: deps.Library,
	}
}

// ListSkills implements Interface. fino:req-fields 1=Keyword,2=Source,4=Tags,5=Uncategorized,3=Category,90=PageSize,91=PageToken
func (s skillhubServer) ListSkills(ctx context.Context, in *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	logs.Ctx(ctx).Debugw("ListSkills")
	if in == nil {
		logs.Ctx(ctx).Warnw("ListSkills rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	filter := repo.SkillFilter{
		Keyword:       strings.TrimSpace(in.GetKeyword()),
		Category:      in.GetCategory(),
		Source:        in.GetSource(),
		Tags:          in.GetTags(),
		Uncategorized: in.GetUncategorized(),
	}

	rows, next, err := s.library.Skills(ctx, filter, int(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, rpcError(ctx, "ListSkills", err)
	}

	var skills []*pb.SkillView
	for _, row := range rows {
		skills = append(skills, s.skillView(row))
	}

	logs.Ctx(ctx).Debugw("ListSkills succeeded", "returned_count", len(skills))

	return &pb.ListSkillsResponse{
		Skills:        skills,
		NextPageToken: next,
	}, nil
}

// GetSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) GetSkill(ctx context.Context, in *pb.GetSkillRequest) (*pb.GetSkillResponse, error) {
	logs.Ctx(ctx).Debugw("GetSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("GetSkill rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	details, err := s.library.Detail(ctx, in.GetSkillId())
	if err != nil {
		return nil, rpcError(ctx, "GetSkill", err)
	}

	logs.Ctx(ctx).Debugw("GetSkill succeeded")

	return &pb.GetSkillResponse{
		Skill: s.skillView(details),
	}, nil
}

// UpdateSkill implements Interface. fino:req-fields 1=SkillId,2=Skill
func (s skillhubServer) UpdateSkill(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	logs.Ctx(ctx).Debugw("UpdateSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("UpdateSkill rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	input := in.GetSkill()
	if input == nil {
		return nil, rpcError(ctx, "UpdateSkill", service.ErrInvalid)
	}

	metadata := service.Metadata{
		Category:     input.GetCategory(),
		Purpose:      input.GetPurpose(),
		Notes:        input.GetNotes(),
		UpstreamName: input.GetUpstreamName(),
		UpstreamURL:  input.GetUpstreamUrl(),
		Tags:         input.GetTags(),
	}

	details, err := s.library.ReplaceMetadata(ctx, in.GetSkillId(), metadata)
	if err != nil {
		return nil, rpcError(ctx, "UpdateSkill", err)
	}

	logs.Ctx(ctx).Infow("UpdateSkill succeeded", "skill_id", in.GetSkillId())

	return &pb.UpdateSkillResponse{
		Skill: s.skillView(details),
	}, nil
}

// DeleteSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) DeleteSkill(ctx context.Context, in *pb.DeleteSkillRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("DeleteSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("DeleteSkill rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if err := s.library.DeleteSkill(ctx, in.GetSkillId()); err != nil {
		return nil, rpcError(ctx, "DeleteSkill", err)
	}

	logs.Ctx(ctx).Infow("DeleteSkill succeeded", "skill_id", in.GetSkillId())

	return &core.Null{}, nil
}

// ListVersions implements Interface. fino:req-fields 1=SkillId,90=PageSize,91=PageToken
func (s skillhubServer) ListVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	logs.Ctx(ctx).Debugw("ListVersions")
	if in == nil {
		logs.Ctx(ctx).Warnw("ListVersions rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	versions, token, err := s.library.Versions(ctx, in.GetSkillId(), int(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, rpcError(ctx, "ListVersions", err)
	}

	var views []*pb.VersionView
	for _, version := range versions {
		views = append(views, s.versionView(version))
	}

	logs.Ctx(ctx).Debugw("ListVersions succeeded", "returned_count", len(views))

	return &pb.ListVersionsResponse{
		Versions:      views,
		NextPageToken: token,
	}, nil
}

// CreateVersion implements Interface. fino:req-fields 1=SkillId,2=Version
func (s skillhubServer) CreateVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	logs.Ctx(ctx).Debugw("CreateVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("CreateVersion rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	input := in.GetVersion()
	if input == nil {
		return nil, rpcError(ctx, "CreateVersion", service.ErrInvalid)
	}

	versionInput := service.VersionInput{
		Notes:          input.GetNotes(),
		ExpectedDigest: input.GetExpectedDigest(),
		MakeDefault:    input.GetMakeDefault(),
	}

	version, err := s.library.CreateVersion(ctx, in.GetSkillId(), input.GetLocalPath(), input.GetSourceRef(), input.GetCommitSha(), versionInput)
	if err != nil {
		return nil, rpcError(ctx, "CreateVersion", err)
	}

	logs.Ctx(ctx).Infow("CreateVersion succeeded", "skill_id", in.GetSkillId(), "version_id", version.ID)

	return &pb.CreateVersionResponse{
		Version: s.versionView(version),
	}, nil
}

// ScanProject implements Interface. fino:req-fields 1=ProjectPath
func (s skillhubServer) ScanProject(ctx context.Context, in *pb.ScanProjectRequest) (*pb.ScanProjectResponse, error) {
	logs.Ctx(ctx).Debugw("ScanProject")
	if in == nil {
		logs.Ctx(ctx).Warnw("ScanProject rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	dir, err := service.LocalDirectory(in.GetProjectPath())
	if err != nil {
		return nil, rpcError(ctx, "ScanProject", err)
	}

	items, err := service.Scan(dir)
	if err != nil {
		return nil, rpcError(ctx, "ScanProject", errcode.WrapPlatformBadRequest(err))
	}

	logs.Ctx(ctx).Debugw("ScanProject succeeded")

	return &pb.ScanProjectResponse{
		Skills: candidateViews(items),
	}, nil
}

// PreviewImport implements Interface. fino:req-fields 1=Source,2=ScanCollection
func (s skillhubServer) PreviewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	logs.Ctx(ctx).Debugw("PreviewImport")
	if in == nil {
		logs.Ctx(ctx).Warnw("PreviewImport rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	source := in.GetSource()
	input := service.Source{
		Repository: source.GetRepositoryUrl(),
		Path:       source.GetSkillPath(),
		Ref:        source.GetRef(),
	}

	result, err := service.PreviewGitHub(ctx, input, in.GetScanCollection())
	if err != nil {
		return nil, rpcError(ctx, "PreviewImport", err)
	}

	logs.Ctx(ctx).Debugw("PreviewImport succeeded")

	return &pb.PreviewImportResponse{
		ResolvedCommitSha: result.Commit,
		Skills:            candidateViews(result.Items),
	}, nil
}

// ImportSkill implements Interface. fino:req-fields 1=RepositoryUrl,2=SourceRef,3=CommitSha,4=SkillPaths,5=Category,6=Tags
func (s skillhubServer) ImportSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	logs.Ctx(ctx).Debugw("ImportSkill")
	if in == nil {
		logs.Ctx(ctx).Warnw("ImportSkill rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	metadata := service.ImportMetadata{Category: in.GetCategory(), Tags: in.GetTags()}
	result, err := s.library.ImportDirectories(ctx, in.GetRepositoryUrl(), in.GetSourceRef(), in.GetCommitSha(), in.GetSkillPaths(), metadata)
	if err != nil {
		return nil, rpcError(ctx, "ImportSkill", err)
	}

	if len(result.Failures) > 0 {
		logs.Ctx(ctx).Warnw("ImportSkill completed with failures", "reason", "partial_failure", "failed_count", len(result.Failures))
	}
	logs.Ctx(ctx).Infow("ImportSkill completed", "added_count", result.Added, "existing_count", result.Existing)

	return importResponse(result), nil
}

// SetDefaultVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) SetDefaultVersion(ctx context.Context, in *pb.SetDefaultVersionRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("SetDefaultVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("SetDefaultVersion rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if err := s.library.SetDefault(ctx, in.GetSkillId(), in.GetVersionId()); err != nil {
		return nil, rpcError(ctx, "SetDefaultVersion", err)
	}

	logs.Ctx(ctx).Infow("SetDefaultVersion succeeded", "skill_id", in.GetSkillId(), "version_id", in.GetVersionId())

	return &core.Null{}, nil
}

// DeleteVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) DeleteVersion(ctx context.Context, in *pb.DeleteVersionRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("DeleteVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("DeleteVersion rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if err := s.library.DeleteVersion(ctx, in.GetSkillId(), in.GetVersionId()); err != nil {
		return nil, rpcError(ctx, "DeleteVersion", err)
	}

	logs.Ctx(ctx).Infow("DeleteVersion succeeded", "skill_id", in.GetSkillId(), "version_id", in.GetVersionId())

	return &core.Null{}, nil
}

// GetVersionContent implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) GetVersionContent(ctx context.Context, in *pb.GetVersionContentRequest) (*pb.GetVersionContentResponse, error) {
	logs.Ctx(ctx).Debugw("GetVersionContent")
	if in == nil {
		logs.Ctx(ctx).Warnw("GetVersionContent rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.library.VersionContents(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, rpcError(ctx, "GetVersionContent", err)
	}

	logs.Ctx(ctx).Debugw("GetVersionContent succeeded")

	return &pb.GetVersionContentResponse{
		Description: result.Description,
		Files:       result.Files,
	}, nil
}

// GetVersionFile implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Path
func (s skillhubServer) GetVersionFile(ctx context.Context, in *pb.GetVersionFileRequest) (*pb.GetVersionFileResponse, error) {
	logs.Ctx(ctx).Debugw("GetVersionFile")
	if in == nil {
		logs.Ctx(ctx).Warnw("GetVersionFile rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.library.VersionFile(ctx, in.GetSkillId(), in.GetVersionId(), in.GetPath())
	if err != nil {
		return nil, rpcError(ctx, "GetVersionFile", err)
	}

	logs.Ctx(ctx).Debugw("GetVersionFile succeeded")

	return &pb.GetVersionFileResponse{
		Text:     result.Text,
		Size:     result.Size,
		Binary:   result.Binary,
		TooLarge: result.TooLarge,
	}, nil
}

// DownloadZip implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) DownloadZip(ctx context.Context, in *pb.DownloadZipRequest) (*pb.DownloadZipResponse, error) {
	logs.Ctx(ctx).Debugw("DownloadZip")
	if in == nil {
		logs.Ctx(ctx).Warnw("DownloadZip rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	row, err := s.library.Version(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, rpcError(ctx, "DownloadZip", err)
	}

	data, err := s.library.ReadArchive(row)
	if err != nil {
		return nil, rpcError(ctx, "DownloadZip", err)
	}

	logs.Ctx(ctx).Debugw("DownloadZip succeeded")

	return &pb.DownloadZipResponse{
		Archive:  &httpbody.HttpBody{Data: data, ContentType: "application/zip"},
		Filename: service.SafeName(row.Name) + ".zip",
	}, nil
}

// SelectLocalDirectory implements Interface. fino:req-fields
func (s skillhubServer) SelectLocalDirectory(ctx context.Context, in *pb.SelectLocalDirectoryRequest) (*pb.SelectLocalDirectoryResponse, error) {
	logs.Ctx(ctx).Debugw("SelectLocalDirectory")
	if in == nil {
		logs.Ctx(ctx).Warnw("SelectLocalDirectory rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	path, err := service.PickDirectory(ctx)
	if err != nil {
		return nil, rpcError(ctx, "SelectLocalDirectory", errcode.WrapPlatformBadRequest(err))
	}

	logs.Ctx(ctx).Debugw("SelectLocalDirectory succeeded")

	return &pb.SelectLocalDirectoryResponse{
		Path: path,
	}, nil
}

// ExportVersionToDirectory implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Directory,4=ChooseDirectory
func (s skillhubServer) ExportVersionToDirectory(ctx context.Context, in *pb.ExportVersionToDirectoryRequest) (*pb.ExportVersionToDirectoryResponse, error) {
	logs.Ctx(ctx).Debugw("ExportVersionToDirectory")
	if in == nil {
		logs.Ctx(ctx).Warnw("ExportVersionToDirectory rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	path, err := s.library.Export(ctx, in.GetSkillId(), in.GetVersionId(), in.GetDirectory(), in.GetChooseDirectory())
	if err != nil {
		return nil, rpcError(ctx, "ExportVersionToDirectory", err)
	}

	logs.Ctx(ctx).Infow("ExportVersionToDirectory succeeded", "skill_id", in.GetSkillId(), "version_id", in.GetVersionId())

	return &pb.ExportVersionToDirectoryResponse{
		Path: path,
	}, nil
}

// RepairVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) RepairVersion(ctx context.Context, in *pb.RepairVersionRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("RepairVersion")
	if in == nil {
		logs.Ctx(ctx).Warnw("RepairVersion rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if err := s.library.Repair(ctx, in.GetSkillId(), in.GetVersionId()); err != nil {
		return nil, rpcError(ctx, "RepairVersion", err)
	}

	logs.Ctx(ctx).Infow("RepairVersion succeeded", "skill_id", in.GetSkillId(), "version_id", in.GetVersionId())

	return &core.Null{}, nil
}

// ListFacets implements Interface. fino:req-fields
func (s skillhubServer) ListFacets(ctx context.Context, in *pb.ListFacetsRequest) (*pb.ListFacetsResponse, error) {
	logs.Ctx(ctx).Debugw("ListFacets")
	if in == nil {
		logs.Ctx(ctx).Warnw("ListFacets rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.library.Facets(ctx)
	if err != nil {
		return nil, rpcError(ctx, "ListFacets", err)
	}

	logs.Ctx(ctx).Debugw("ListFacets succeeded")

	return &pb.ListFacetsResponse{
		Categories: result.Categories,
		Sources:    result.Sources,
		Tags:       result.Tags,
	}, nil
}

// BatchUpdateSkillMetadata implements Interface. fino:req-fields 1=Changes
func (s skillhubServer) BatchUpdateSkillMetadata(ctx context.Context, in *pb.BatchUpdateSkillMetadataRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("BatchUpdateSkillMetadata")
	if in == nil {
		logs.Ctx(ctx).Warnw("BatchUpdateSkillMetadata rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	input := make([]service.MetadataChange, 0, len(in.GetChanges()))
	for _, change := range in.GetChanges() {
		if change == nil {
			return nil, rpcError(ctx, "BatchUpdateSkillMetadata", service.ErrInvalid)
		}
		input = append(input, service.MetadataChange{
			SkillID: change.GetSkillId(), Revision: change.GetRevision(), Category: change.Category, Purpose: change.Purpose,
			AddTags: change.GetAddTags(), RemoveTags: change.GetRemoveTags(),
		})
	}

	if err := s.library.Organize(ctx, input); err != nil {
		return nil, rpcError(ctx, "BatchUpdateSkillMetadata", err)
	}

	logs.Ctx(ctx).Infow("BatchUpdateSkillMetadata succeeded", "updated_count", len(input))

	return &core.Null{}, nil
}

// UploadZip implements Interface. fino:req-fields 1=File
func (s skillhubServer) UploadZip(ctx context.Context, in *pb.UploadZipRequest) (*pb.UploadZipResponse, error) {
	logs.Ctx(ctx).Debugw("UploadZip")
	if in == nil {
		logs.Ctx(ctx).Warnw("UploadZip rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.library.StageArchive(in.GetFile())
	if err != nil {
		return nil, rpcError(ctx, "UploadZip", err)
	}

	logs.Ctx(ctx).Infow("UploadZip succeeded", "candidate_count", len(result.Items))

	return &pb.UploadZipResponse{
		UploadId: result.ID,
		Skills:   candidateViews(result.Items),
	}, nil
}

// CommitUpload implements Interface. fino:req-fields 1=UploadId,2=SkillPaths,3=TargetSkillId,4=MakeDefault
func (s skillhubServer) CommitUpload(ctx context.Context, in *pb.CommitUploadRequest) (*pb.ImportSkillResponse, error) {
	logs.Ctx(ctx).Debugw("CommitUpload")
	if in == nil {
		logs.Ctx(ctx).Warnw("CommitUpload rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	result, err := s.library.ImportArchive(ctx, in.GetUploadId(), in.GetTargetSkillId(), in.GetSkillPaths(), in.GetMakeDefault())
	if err != nil {
		return nil, rpcError(ctx, "CommitUpload", err)
	}

	if len(result.Failures) > 0 {
		logs.Ctx(ctx).Warnw("CommitUpload completed with failures", "reason", "partial_failure", "failed_count", len(result.Failures))
	}
	logs.Ctx(ctx).Infow("CommitUpload completed", "added_count", result.Added, "existing_count", result.Existing)

	return importResponse(result), nil
}

// CompareSkills implements Interface. fino:req-fields 1=Left,2=Right
func (s skillhubServer) CompareSkills(ctx context.Context, in *pb.CompareSkillsRequest) (*pb.CompareSkillsResponse, error) {
	logs.Ctx(ctx).Debugw("CompareSkills")
	if in == nil {
		logs.Ctx(ctx).Warnw("CompareSkills rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	diff, err := s.library.Compare(ctx, location(in.GetLeft()), location(in.GetRight()))
	if err != nil {
		return nil, rpcError(ctx, "CompareSkills", err)
	}

	logs.Ctx(ctx).Debugw("CompareSkills succeeded", "changed_count", len(diff.Files))

	return comparisonView(diff), nil
}

// GetInstallCommand implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Global
func (s skillhubServer) GetInstallCommand(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
	logs.Ctx(ctx).Debugw("GetInstallCommand")
	if in == nil {
		logs.Ctx(ctx).Warnw("GetInstallCommand rejected", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	command, err := s.library.InstallCommand(ctx, in.GetSkillId(), in.GetVersionId(), in.GetGlobal())
	if err != nil {
		return nil, rpcError(ctx, "GetInstallCommand", err)
	}

	logs.Ctx(ctx).Debugw("GetInstallCommand succeeded")

	return &pb.GetInstallCommandResponse{
		Command: command,
	}, nil
}

// BatchDeleteSkills implements Interface. fino:req-fields 1=SkillIds
func (s skillhubServer) BatchDeleteSkills(ctx context.Context, in *pb.BatchDeleteSkillsRequest) (*core.Null, error) {
	logs.Ctx(ctx).Debugw("BatchDeleteSkills")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid batch_delete_skills request", "reason", "request_missing")
		return nil, errcode.NewSkillhubRequestMissing()
	}

	if err := s.library.BatchDeleteSkills(ctx, in.GetSkillIds()); err != nil {
		return nil, rpcError(ctx, "BatchDeleteSkills", err)
	}

	logs.Ctx(ctx).Infow("BatchDeleteSkills succeeded", "requested_count", len(in.GetSkillIds()))
	return &core.Null{}, nil
}
