package app

import (
	"context"
	"fmt"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

type comparisonSource struct {
	path    string
	commit  string
	cleanup func()
}

func (s skillhubServer) gitHubVersion(ctx context.Context, skill *model.Skill, version *model.SkillVersion) (comparisonSource, error) {
	if skill.RepositoryUrl() == "" || version.CommitSha() == "" {
		return comparisonSource{}, fmt.Errorf("version has no pinned commit")
	}
	path, commit, cleanup, err := service.CheckoutGitHub(ctx, service.GitHubSource{
		RepositoryURL: skill.RepositoryUrl(), SkillPath: skill.RepositoryPath(), Ref: version.CommitSha(),
	})
	return comparisonSource{path: path, commit: commit, cleanup: cleanup}, err
}

func (s skillhubServer) compareSource(ctx context.Context, location *pb.SkillLocation) (comparisonSource, error) {
	if path := location.GetLocalPath(); path != "" {
		return comparisonSource{path: path}, nil
	}
	if source := location.GetGithub(); source != nil {
		path, commit, cleanup, err := service.CheckoutGitHub(ctx, githubSource(source))
		return comparisonSource{path: path, commit: commit, cleanup: cleanup}, err
	}
	if id := location.GetCurrentSkillId(); id != "" {
		skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(id))
		if err != nil {
			return comparisonSource{}, err
		}
		versions, err := s.skillhubSvc.ListSkillVersions(ctx, model.ListSkillVersionQuery{Filter: model.SkillVersionFilter{SkillID: id}, PageSize: 1, SkipTotalCount: true})
		if err != nil {
			return comparisonSource{}, err
		}
		if len(versions.Items) == 0 {
			return comparisonSource{}, fmt.Errorf("skill has no versions")
		}
		return s.gitHubVersion(ctx, skill, versions.Items[0])
	}
	if id := location.GetVersionId(); id != "" {
		version, err := s.skillhubSvc.GetSkillVersion(ctx, model.SkillVersionID(id))
		if err != nil {
			return comparisonSource{}, err
		}
		skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(version.SkillId()))
		if err != nil {
			return comparisonSource{}, err
		}
		return s.gitHubVersion(ctx, skill, version)
	}
	return comparisonSource{}, fmt.Errorf("source missing")
}

func (s skillhubServer) compareSkills(ctx context.Context, in *pb.CompareSkillsRequest) (*pb.CompareSkillsResponse, error) {
	left, err := s.compareSource(ctx, in.GetLeft())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	if left.cleanup != nil {
		defer left.cleanup()
	}
	right, err := s.compareSource(ctx, in.GetRight())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	if right.cleanup != nil {
		defer right.cleanup()
	}
	changes, leftDigest, rightDigest, err := service.CompareDirectories(left.path, right.path)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	files := make([]*pb.FileChangeView, 0, len(changes))
	for _, change := range changes {
		kind := pb.FileChangeKind_FILE_CHANGE_KIND_MODIFIED
		if change.Kind == "added" {
			kind = pb.FileChangeKind_FILE_CHANGE_KIND_ADDED
		}
		if change.Kind == "deleted" {
			kind = pb.FileChangeKind_FILE_CHANGE_KIND_DELETED
		}
		files = append(files, &pb.FileChangeView{Path: change.Path, Kind: kind, Binary: change.Binary, LeftText: change.LeftText, RightText: change.RightText})
	}
	return &pb.CompareSkillsResponse{Files: files, LeftDigest: leftDigest, RightDigest: rightDigest, LeftCommitSha: left.commit, RightCommitSha: right.commit}, nil
}
