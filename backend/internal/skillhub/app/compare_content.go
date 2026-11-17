package app

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func location(input *pb.SkillLocation) service.Location {
	result := service.Location{
		SkillID: input.GetCurrentSkillId(), VersionID: input.GetVersionId(), LocalPath: input.GetLocalPath(),
	}
	if github := input.GetGithub(); github != nil {
		result.GitHub = &service.Source{Repository: github.GetRepositoryUrl(), Ref: github.GetRef(), Path: github.GetSkillPath()}
	}
	return result
}

func (s skillhubServer) compare(ctx context.Context, in *pb.CompareSkillsRequest) (*pb.CompareSkillsResponse, error) {
	diff, err := s.library.Compare(ctx, location(in.GetLeft()), location(in.GetRight()))
	if err != nil {
		return nil, failure(err)
	}
	result := &pb.CompareSkillsResponse{
		LeftDigest: diff.LeftDigest, RightDigest: diff.RightDigest,
		LeftCommitSha: diff.LeftCommit, RightCommitSha: diff.RightCommit,
	}
	for _, file := range diff.Files {
		kind := pb.FileChangeKind_FILE_CHANGE_KIND_MODIFIED
		switch file.Kind {
		case service.Added:
			kind = pb.FileChangeKind_FILE_CHANGE_KIND_ADDED
		case service.Deleted:
			kind = pb.FileChangeKind_FILE_CHANGE_KIND_DELETED
		}
		result.Files = append(result.Files, &pb.FileChangeView{
			Path: file.Path, Kind: kind, Binary: file.Binary, LeftText: file.LeftText, RightText: file.RightText, Detail: file.Detail,
		})
	}
	return result, nil
}
