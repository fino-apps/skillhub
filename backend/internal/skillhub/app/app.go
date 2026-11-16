package app

import (
	"context"
	"sync"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

type ServiceDeps struct {
	SkillhubSvc service.ISkillhubService
	Catalog     repo.CatalogRepo
	Snapshots   *service.SnapshotStore
	ExportDir   string
	UploadDir   string
	AIModel     string
}

type skillhubServer struct {
	pb.UnimplementedSkillhubServiceServer

	skillhubSvc service.ISkillhubService
	catalog     repo.CatalogRepo
	snapshots   *service.SnapshotStore
	exportDir   string
	uploadDir   string
	aiModel     string
	files       *sync.RWMutex
	aiQueue     chan struct{}
	uploads     *sync.Mutex
}

// NewService wires the generated transport server to the application service.
func NewService(deps ServiceDeps) *skillhubServer {
	return &skillhubServer{
		skillhubSvc: deps.SkillhubSvc,
		catalog:     deps.Catalog,
		snapshots:   deps.Snapshots,
		exportDir:   deps.ExportDir,
		uploadDir:   deps.UploadDir,
		aiModel:     deps.AIModel,
		files:       &sync.RWMutex{},
		aiQueue:     make(chan struct{}, 1),
		uploads:     &sync.Mutex{},
	}
}

// ListSkills implements Interface. fino:req-fields 1=Keyword,2=Source,4=Tags,5=Uncategorized,3=Category,90=PageSize,91=PageToken
func (s skillhubServer) ListSkills(ctx context.Context, in *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	return s.listSkills(ctx, in)
}

// GetSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) GetSkill(ctx context.Context, in *pb.GetSkillRequest) (*pb.GetSkillResponse, error) {
	return s.getSkill(ctx, in)
}

// UpdateSkill implements Interface. fino:req-fields 1=SkillId,2=Skill
func (s skillhubServer) UpdateSkill(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	return s.updateSkill(ctx, in)
}

// DeleteSkill implements Interface. fino:req-fields 1=SkillId
func (s skillhubServer) DeleteSkill(ctx context.Context, in *pb.DeleteSkillRequest) (*core.Null, error) {
	s.files.Lock()
	defer s.files.Unlock()
	id := model.SkillID(in.GetSkillId())
	if err := s.catalog.DeleteSkill(ctx, id); err != nil {
		return nil, catalogError(err)
	}
	s.removeSnapshot(string(id), "")
	return &core.Null{}, nil
}

// ListVersions implements Interface. fino:req-fields 1=SkillId,90=PageSize,91=PageToken
func (s skillhubServer) ListVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	return s.listVersions(ctx, in)
}

// CreateVersion implements Interface. fino:req-fields 1=SkillId,2=Version
func (s skillhubServer) CreateVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	return s.createVersion(ctx, in)
}

// ScanProject implements Interface. fino:req-fields 1=ProjectPath
func (s skillhubServer) ScanProject(ctx context.Context, in *pb.ScanProjectRequest) (*pb.ScanProjectResponse, error) {
	items, err := service.ScanProject(in.GetProjectPath())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.ScanProjectResponse{Skills: candidates(items)}, nil
}

// PreviewImport implements Interface. fino:req-fields 1=Source,2=ScanCollection
func (s skillhubServer) PreviewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	return s.previewImport(ctx, in)
}

// ImportSkill implements Interface. fino:req-fields 1=RepositoryUrl,2=SourceRef,3=CommitSha,4=SkillPaths
func (s skillhubServer) ImportSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	return s.importSkill(ctx, in)
}

// CompareSkills implements Interface. fino:req-fields 1=Left,2=Right
func (s skillhubServer) CompareSkills(ctx context.Context, in *pb.CompareSkillsRequest) (*pb.CompareSkillsResponse, error) {
	if !validSkillLocation(in.GetLeft()) || !validSkillLocation(in.GetRight()) {
		return nil, errcode.NewPlatformBadRequest()
	}
	s.files.RLock()
	defer s.files.RUnlock()
	return s.compareSkills(ctx, in)
}

// GetInstallCommand implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Global
func (s skillhubServer) GetInstallCommand(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
	s.files.RLock()
	defer s.files.RUnlock()
	return s.getInstallCommand(ctx, in)
}

// SetDefaultVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) SetDefaultVersion(ctx context.Context, in *pb.SetDefaultVersionRequest) (*core.Null, error) {
	return s.setDefaultVersion(ctx, in)
}

// DeleteVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) DeleteVersion(ctx context.Context, in *pb.DeleteVersionRequest) (*core.Null, error) {
	return s.deleteVersion(ctx, in)
}

// GetVersionContent implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) GetVersionContent(ctx context.Context, in *pb.GetVersionContentRequest) (*pb.GetVersionContentResponse, error) {
	return s.getVersionContent(ctx, in)
}

// GetVersionFile implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Path
func (s skillhubServer) GetVersionFile(ctx context.Context, in *pb.GetVersionFileRequest) (*pb.GetVersionFileResponse, error) {
	return s.getVersionFile(ctx, in)
}

// SelectLocalDirectory implements Interface. fino:req-fields
func (s skillhubServer) SelectLocalDirectory(ctx context.Context, in *pb.SelectLocalDirectoryRequest) (*pb.SelectLocalDirectoryResponse, error) {
	path, err := service.SelectDirectory(ctx)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.SelectLocalDirectoryResponse{Path: path}, nil
}

// ExportVersionToDirectory implements Interface. fino:req-fields 1=SkillId,2=VersionId,3=Directory,4=ChooseDirectory
func (s skillhubServer) ExportVersionToDirectory(ctx context.Context, in *pb.ExportVersionToDirectoryRequest) (*pb.ExportVersionToDirectoryResponse, error) {
	return s.exportVersionToDirectory(ctx, in)
}

// RepairVersion implements Interface. fino:req-fields 1=SkillId,2=VersionId
func (s skillhubServer) RepairVersion(ctx context.Context, in *pb.RepairVersionRequest) (*core.Null, error) {
	return s.repairVersion(ctx, in)
}

// ListFacets implements Interface. fino:req-fields
func (s skillhubServer) ListFacets(ctx context.Context, in *pb.ListFacetsRequest) (*pb.ListFacetsResponse, error) {
	categories, sources, tags, err := s.catalog.Facets(ctx)
	if err != nil {
		return nil, catalogError(err)
	}
	return &pb.ListFacetsResponse{Categories: categories, Sources: sources, Tags: tags, AiEnabled: s.aiModel != ""}, nil
}

// BatchUpdateSkillMetadata implements Interface. fino:req-fields 1=Changes
func (s skillhubServer) BatchUpdateSkillMetadata(ctx context.Context, in *pb.BatchUpdateSkillMetadataRequest) (*core.Null, error) {
	return s.batchUpdateSkillMetadata(ctx, in)
}

// CommitUpload implements Interface. fino:req-fields 1=UploadId,2=SkillPaths,3=TargetSkillId,4=MakeDefault
func (s skillhubServer) CommitUpload(ctx context.Context, in *pb.CommitUploadRequest) (*pb.ImportSkillResponse, error) {
	return s.commitUpload(ctx, in)
}

// CreateOrganizationTask implements Interface. fino:req-fields 1=SkillIds,2=Uncategorized
func (s skillhubServer) CreateOrganizationTask(ctx context.Context, in *pb.CreateOrganizationTaskRequest) (*pb.OrganizationTaskView, error) {
	return s.createOrganizationTask(ctx, in)
}

// GetOrganizationTask implements Interface. fino:req-fields 1=TaskId
func (s skillhubServer) GetOrganizationTask(ctx context.Context, in *pb.GetOrganizationTaskRequest) (*pb.OrganizationTaskView, error) {
	return s.getOrganizationTask(ctx, in)
}
