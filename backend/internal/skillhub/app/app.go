package app

import (
	"context"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
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
	return s.querySkills(ctx, in)
}

// GetSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) GetSkill(ctx context.Context, in *pb.GetSkillRequest) (*pb.GetSkillResponse, error) {
	view, err := s.readSkill(ctx, in.GetSkillId())
	if err != nil {
		return nil, err
	}
	return &pb.GetSkillResponse{Skill: view}, nil
}

// UpdateSkill implements Interface. fino:req-fields 1=SkillId,2=Skill
func (s skillhubServer) UpdateSkill(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	return s.replaceMetadata(ctx, in)
}

// DeleteSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) DeleteSkill(ctx context.Context, in *pb.DeleteSkillRequest) (*core.Null, error) {
	return empty(failure(s.library.DeleteSkill(ctx, in.GetSkillId())))
}

// ListVersions implements Interface. fino:req-fields 1=SkillId,90=PageSize,91=PageToken
func (s skillhubServer) ListVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	return s.queryVersions(ctx, in)
}

// CreateVersion implements Interface. fino:req-fields 1=SkillId,2=Version
func (s skillhubServer) CreateVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	return s.saveVersion(ctx, in)
}

// ScanProject implements Interface. fino:req-fields 1=ProjectPath
func (s skillhubServer) ScanProject(ctx context.Context, in *pb.ScanProjectRequest) (*pb.ScanProjectResponse, error) {
	dir, err := service.LocalDirectory(in.GetProjectPath())
	if err != nil {
		return nil, failure(err)
	}
	items, err := service.Scan(dir)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.ScanProjectResponse{Skills: candidateViews(items)}, nil
}

// PreviewImport implements Interface. fino:req-fields 1=Source,2=ScanCollection
func (s skillhubServer) PreviewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	return previewGitHub(ctx, in)
}

// ImportSkill implements Interface. fino:req-fields 1=RepositoryUrl,2=SourceRef,3=CommitSha,4=SkillPaths
func (s skillhubServer) ImportSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	return s.importDirectories(ctx, in)
}

// SetDefaultVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) SetDefaultVersion(ctx context.Context, in *pb.SetDefaultVersionRequest) (*core.Null, error) {
	return empty(failure(s.library.SetDefault(ctx, in.GetSkillId(), in.GetVersionId())))
}

// DeleteVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) DeleteVersion(ctx context.Context, in *pb.DeleteVersionRequest) (*core.Null, error) {
	return empty(failure(s.library.DeleteVersion(ctx, in.GetSkillId(), in.GetVersionId())))
}

// GetVersionContent implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) GetVersionContent(ctx context.Context, in *pb.GetVersionContentRequest) (*pb.GetVersionContentResponse, error) {
	return s.versionContents(ctx, in.GetSkillId(), in.GetVersionId())
}

// GetVersionFile implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Path
func (s skillhubServer) GetVersionFile(ctx context.Context, in *pb.GetVersionFileRequest) (*pb.GetVersionFileResponse, error) {
	return s.versionFile(ctx, in.GetSkillId(), in.GetVersionId(), in.GetPath())
}

// DownloadZip implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) DownloadZip(ctx context.Context, in *pb.DownloadZipRequest) (*pb.DownloadZipResponse, error) {
	return s.download(ctx, in.GetSkillId(), in.GetVersionId())
}

// SelectLocalDirectory implements Interface. fino:req-fields
func (s skillhubServer) SelectLocalDirectory(ctx context.Context, in *pb.SelectLocalDirectoryRequest) (*pb.SelectLocalDirectoryResponse, error) {
	path, err := service.PickDirectory(ctx)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.SelectLocalDirectoryResponse{Path: path}, nil
}

// ExportVersionToDirectory implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Directory,4=ChooseDirectory
func (s skillhubServer) ExportVersionToDirectory(ctx context.Context, in *pb.ExportVersionToDirectoryRequest) (*pb.ExportVersionToDirectoryResponse, error) {
	return s.exportVersion(ctx, in)
}

// RepairVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) RepairVersion(ctx context.Context, in *pb.RepairVersionRequest) (*core.Null, error) {
	return empty(failure(s.library.Repair(ctx, in.GetSkillId(), in.GetVersionId())))
}

// ListFacets implements Interface. fino:req-fields
func (s skillhubServer) ListFacets(ctx context.Context, in *pb.ListFacetsRequest) (*pb.ListFacetsResponse, error) {
	return s.facets(ctx)
}

// BatchUpdateSkillMetadata implements Interface. fino:req-fields 1=Changes
func (s skillhubServer) BatchUpdateSkillMetadata(ctx context.Context, in *pb.BatchUpdateSkillMetadataRequest) (*core.Null, error) {
	return empty(s.organize(ctx, in.GetChanges()))
}

// UploadZip implements Interface. fino:req-fields 1=File
func (s skillhubServer) UploadZip(ctx context.Context, in *pb.UploadZipRequest) (*pb.UploadZipResponse, error) {
	return s.stageArchive(in.GetFile())
}

// CommitUpload implements Interface. fino:req-fields 1=UploadId,2=SkillPaths,3=TargetSkillId,4=MakeDefault
func (s skillhubServer) CommitUpload(ctx context.Context, in *pb.CommitUploadRequest) (*pb.ImportSkillResponse, error) {
	return s.importArchive(ctx, in)
}

// CompareSkills implements Interface. fino:req-fields 1=Left,2=Right
func (s skillhubServer) CompareSkills(ctx context.Context, in *pb.CompareSkillsRequest) (*pb.CompareSkillsResponse, error) {
	return s.compare(ctx, in)
}

// GetInstallCommand implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Global
func (s skillhubServer) GetInstallCommand(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
	return s.installation(ctx, in)
}

func empty(err error) (*core.Null, error) {
	if err != nil {
		return nil, err
	}
	return &core.Null{}, nil
}
