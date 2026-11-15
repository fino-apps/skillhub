package app

import (
	"context"
	"errors"

	"github.com/fino-io/skillhub/backend/internal/skillhub/app/mapper"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func githubSource(source *pb.GitHubSource) service.GitHubSource {
	return service.GitHubSource{RepositoryURL: source.GetRepositoryUrl(), SkillPath: source.GetSkillPath(), Ref: source.GetRef()}
}

func (s skillhubServer) previewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	if in.GetTargetSkillId() != "" {
		return nil, errcode.NewPlatformBadRequest()
	}
	var files []service.ImportFile
	var digest, commit string
	var err error
	if source := in.GetSource().GetGithub(); source != nil {
		var preview service.GitHubPreview
		preview, err = service.PreviewGitHub(ctx, githubSource(source))
		files, digest, commit = preview.Files, preview.Digest, preview.Commit
	} else if in.GetSource().GetLocalPath() != "" {
		files, digest, err = service.PreviewLocalSkill(in.GetSource().GetLocalPath())
	} else {
		return nil, errcode.NewPlatformBadRequest()
	}
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	result := make([]*pb.FileView, 0, len(files))
	for _, file := range files {
		result = append(result, &pb.FileView{Path: file.Path, Size: file.Size, Digest: file.Digest, Sensitive: file.Sensitive})
	}
	return &pb.PreviewImportResponse{Files: result, SourceDigest: digest, ResolvedCommitSha: commit}, nil
}

func (s skillhubServer) importSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	if s.intake == nil || in.GetTargetSkillId() != "" || in.GetVersion() == nil {
		return nil, errcode.NewPlatformBadRequest()
	}
	var skill *model.Skill
	var version *model.SkillVersion
	var err error
	if source := in.GetSource().GetGithub(); source != nil {
		skill, version, err = s.intake.ImportPublic(ctx, githubSource(source), in.GetExpectedSourceDigest(), in.GetVersion().GetCommitSha(), in.GetVersion().GetVersion(), in.GetVersion().GetNotes())
	} else if in.GetSource().GetLocalPath() != "" {
		skill, version, err = s.intake.ImportLocal(ctx, in.GetSource().GetLocalPath(), in.GetExpectedSourceDigest(), in.GetIncludedFiles(), in.GetVersion().GetVersion(), in.GetVersion().GetNotes())
	} else {
		return nil, errcode.NewPlatformBadRequest()
	}
	if errors.Is(err, repo.ErrSkillExists) {
		return nil, errcode.WrapPlatformConflict(err)
	}
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.ImportSkillResponse{Skill: mapper.ToSkillView(skill), Version: &pb.VersionView{
		Id: string(version.ID()), SkillId: version.SkillId(), Version: version.Version(), Notes: version.Notes(), Digest: version.Digest(), SourceRef: version.SourceRef(), CommitSha: version.CommitSha(), CreateTime: version.CreateTime(),
	}}, nil
}

func (s skillhubServer) listVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	if _, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId())); err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	result, err := s.skillhubSvc.ListSkillVersions(ctx, model.ListSkillVersionQuery{
		Filter: model.SkillVersionFilter{SkillID: in.GetSkillId()}, PageSize: int(in.GetPageSize()), PageToken: in.GetPageToken(), SkipTotalCount: true,
	})
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	versions := make([]*pb.VersionView, 0, len(result.Items))
	for _, version := range result.Items {
		versions = append(versions, &pb.VersionView{Id: string(version.ID()), SkillId: version.SkillId(), Version: version.Version(), Notes: version.Notes(), Digest: version.Digest(), SourceRef: version.SourceRef(), CommitSha: version.CommitSha(), CreateTime: version.CreateTime()})
	}
	return &pb.ListVersionsResponse{Versions: versions, NextPageToken: result.NextPageToken}, nil
}
