package app

import (
	"context"
	"strings"

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
	skills, commit, err := service.InspectGitHub(ctx, githubSource(in.GetSource()), in.GetScanCollection())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	items := make([]*pb.SkillCandidate, 0, len(skills))
	for _, skill := range skills {
		items = append(items, &pb.SkillCandidate{Path: skill.Path, Name: skill.Name, Description: skill.Description})
	}
	return &pb.PreviewImportResponse{Skills: items, ResolvedCommitSha: commit}, nil
}

func (s skillhubServer) importSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	ref := strings.TrimSpace(in.GetSourceRef())
	paths := in.GetSkillPaths()
	if len(paths) == 0 || len(paths) > 100 {
		return nil, errcode.NewPlatformBadRequest()
	}
	if in.GetRepositoryUrl() == "" {
		if ref != "" || in.GetCommitSha() != "" {
			return nil, errcode.NewPlatformBadRequest()
		}
		return s.importLocalSkills(ctx, paths)
	}
	if ref == "" || len(ref) > 200 {
		return nil, errcode.NewPlatformBadRequest()
	}
	repository, err := service.NormalizeRepository(in.GetRepositoryUrl())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	commit := in.GetCommitSha()
	skills, err := service.ReadPinnedGitHubSkills(ctx, repository, commit, paths)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	entries := make([]repo.CatalogEntry, 0, len(skills))
	seen := make(map[string]bool, len(skills))
	for _, item := range skills {
		if seen[item.Path] {
			continue
		}
		seen[item.Path] = true
		skill, err := model.NewSkill(model.CreateSkillInput{
			Name: item.Name, Description: item.Description, RepositoryUrl: repository, RepositoryPath: item.Path,
		})
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		version, err := pinnedVersion(string(skill.ID()), item.Name, ref, commit, "")
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		entries = append(entries, repo.CatalogEntry{Skill: skill, Version: version})
	}
	added, err := s.catalog.AddSkills(ctx, entries)
	if err != nil {
		return nil, errcode.WrapCreateSkillhubFailed(err)
	}
	return &pb.ImportSkillResponse{AddedCount: int32(len(added)), ExistingCount: int32(len(entries) - len(added))}, nil
}

func pinnedVersion(skillID, name, ref, commit, notes string) (*model.SkillVersion, error) {
	return model.NewSkillVersion(model.CreateSkillVersionInput{
		SkillId: skillID, Name: name, SourceRef: ref, CommitSha: commit, Notes: notes,
	})
}

func versionView(version *model.SkillVersion) *pb.VersionView {
	return &pb.VersionView{
		Id: string(version.ID()), SkillId: version.SkillId(), Name: version.Name(),
		Notes: version.Notes(), SourceRef: version.SourceRef(), CommitSha: version.CommitSha(),
		ContentDigest: version.ContentDigest(), CreateTime: version.CreateTime(),
	}
}

func (s skillhubServer) createVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId()))
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	input := in.GetVersion()
	if input == nil {
		return nil, errcode.NewPlatformBadRequest()
	}
	if skill.LocalPath() != "" {
		if input.GetSourceRef() != "" || input.GetCommitSha() != "" {
			return nil, errcode.NewPlatformBadRequest()
		}
		return s.captureVersion(ctx, skill, input)
	}
	if input.GetExpectedDigest() != "" {
		return nil, errcode.NewPlatformBadRequest()
	}
	ref, commit := strings.TrimSpace(input.GetSourceRef()), input.GetCommitSha()
	if ref == "" || len(ref) > 200 {
		return nil, errcode.NewPlatformBadRequest()
	}
	items, err := service.ReadPinnedGitHubSkills(ctx, skill.RepositoryUrl(), commit, []string{skill.RepositoryPath()})
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	version, err := pinnedVersion(in.GetSkillId(), items[0].Name, ref, commit, input.GetNotes())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return s.saveVersion(ctx, version, items[0].Description)
}

func (s skillhubServer) saveVersion(ctx context.Context, version *model.SkillVersion, description string) (*pb.CreateVersionResponse, error) {
	created, err := s.catalog.AddVersion(ctx, version, description)
	if err != nil {
		return nil, errcode.WrapCreateSkillhubFailed(err)
	}
	if !created {
		return nil, errcode.NewPlatformConflict()
	}
	return &pb.CreateVersionResponse{Version: versionView(version)}, nil
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
		versions = append(versions, versionView(version))
	}
	return &pb.ListVersionsResponse{Versions: versions, NextPageToken: result.NextPageToken}, nil
}
