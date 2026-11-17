package app

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func candidateViews(items []service.Candidate) []*pb.SkillCandidate {
	result := make([]*pb.SkillCandidate, 0, len(items))
	for _, item := range items {
		result = append(result, &pb.SkillCandidate{Path: item.Path, Name: item.Name, Description: item.Description})
	}
	return result
}

func previewGitHub(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	source := in.GetSource()
	result, err := service.PreviewGitHub(ctx, service.Source{
		Repository: source.GetRepositoryUrl(), Path: source.GetSkillPath(), Ref: source.GetRef(),
	}, in.GetScanCollection())
	if err != nil {
		return nil, failure(err)
	}
	return &pb.PreviewImportResponse{ResolvedCommitSha: result.Commit, Skills: candidateViews(result.Items)}, nil
}

func (s skillhubServer) importDirectories(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	result, err := s.library.ImportDirectories(ctx, in.GetRepositoryUrl(), in.GetSourceRef(), in.GetCommitSha(), in.GetSkillPaths())
	return importResponse(result, err)
}

func (s skillhubServer) stageArchive(file []byte) (*pb.UploadZipResponse, error) {
	result, err := s.library.StageArchive(file)
	if err != nil {
		return nil, failure(err)
	}
	return &pb.UploadZipResponse{UploadId: result.ID, Skills: candidateViews(result.Items)}, nil
}

func (s skillhubServer) importArchive(ctx context.Context, in *pb.CommitUploadRequest) (*pb.ImportSkillResponse, error) {
	result, err := s.library.ImportArchive(ctx, in.GetUploadId(), in.GetTargetSkillId(), in.GetSkillPaths(), in.GetMakeDefault())
	return importResponse(result, err)
}

func importResponse(result *service.ImportResult, err error) (*pb.ImportSkillResponse, error) {
	if err != nil {
		return nil, failure(err)
	}
	response := &pb.ImportSkillResponse{AddedCount: int32(result.Added), ExistingCount: int32(result.Existing)}
	for _, item := range result.Failures {
		response.Failures = append(response.Failures, &pb.ImportFailure{Path: item.Path, Message: failure(item.Err).Error()})
	}
	return response, nil
}

func (s skillhubServer) saveVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	input := in.GetVersion()
	if input == nil {
		return nil, errcode.NewPlatformBadRequest()
	}
	version, err := s.library.CreateVersion(ctx, in.GetSkillId(), input.GetLocalPath(), input.GetSourceRef(), input.GetCommitSha(), service.VersionInput{
		Notes: input.GetNotes(), ExpectedDigest: input.GetExpectedDigest(), MakeDefault: input.GetMakeDefault(),
	})
	if err != nil {
		return nil, failure(err)
	}
	return &pb.CreateVersionResponse{Version: s.versionView(version)}, nil
}
