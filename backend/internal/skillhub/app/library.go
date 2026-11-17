package app

import (
	"context"
	"errors"
	"strings"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func failure(err error) error {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return errcode.NewPlatformNotFound()
	case errors.Is(err, repo.ErrPageToken), errors.Is(err, service.ErrInvalid):
		return errcode.WrapPlatformBadRequest(err)
	case errors.Is(err, service.ErrConflict):
		return errcode.WrapPlatformConflict(err)
	default:
		return err
	}
}

func (s skillhubServer) skillView(details service.SkillDetails) *pb.SkillView {
	row := details.Skill
	view := &pb.SkillView{
		Id: string(row.ID), Name: row.Name, Description: row.Description, Category: row.Category, Notes: row.Notes,
		Tags: details.Tags, RepositoryUrl: row.RepositoryUrl, RepositoryPath: row.RepositoryPath,
		LocalPath: row.LocalPath, ImportKind: row.ImportKind, Purpose: row.Purpose,
		UpstreamName: row.UpstreamName, UpstreamUrl: row.UpstreamUrl, CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
	}
	if details.Default != nil {
		view.DefaultVersion = s.versionView(*details.Default)
	}
	return view
}

func (s skillhubServer) querySkills(ctx context.Context, in *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	rows, next, err := s.library.Skills(ctx, repo.SkillFilter{
		Keyword: strings.TrimSpace(in.GetKeyword()), Category: in.GetCategory(), Source: in.GetSource(),
		Tags: in.GetTags(), Uncategorized: in.GetUncategorized(),
	}, int(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, failure(err)
	}
	result := &pb.ListSkillsResponse{NextPageToken: next}
	for _, row := range rows {
		result.Skills = append(result.Skills, s.skillView(row))
	}
	return result, nil
}

func (s skillhubServer) readSkill(ctx context.Context, id string) (*pb.SkillView, error) {
	details, err := s.library.Detail(ctx, id)
	if err != nil {
		return nil, failure(err)
	}
	return s.skillView(details), nil
}

func (s skillhubServer) replaceMetadata(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	input := in.GetSkill()
	if input == nil {
		return nil, errcode.NewPlatformBadRequest()
	}
	details, err := s.library.ReplaceMetadata(ctx, in.GetSkillId(), service.Metadata{
		Category: input.GetCategory(), Purpose: input.GetPurpose(), Notes: input.GetNotes(),
		UpstreamName: input.GetUpstreamName(), UpstreamURL: input.GetUpstreamUrl(), Tags: input.GetTags(),
	})
	if err != nil {
		return nil, failure(err)
	}
	return &pb.UpdateSkillResponse{Skill: s.skillView(details)}, nil
}

func (s skillhubServer) organize(ctx context.Context, changes []*pb.SkillMetadataChange) error {
	input := make([]service.MetadataChange, 0, len(changes))
	for _, change := range changes {
		if change == nil {
			return errcode.NewPlatformBadRequest()
		}
		input = append(input, service.MetadataChange{
			SkillID: change.GetSkillId(), Revision: change.GetRevision(), Category: change.Category, Purpose: change.Purpose,
			AddTags: change.GetAddTags(), RemoveTags: change.GetRemoveTags(),
		})
	}
	return failure(s.library.Organize(ctx, input))
}

func (s skillhubServer) facets(ctx context.Context) (*pb.ListFacetsResponse, error) {
	result, err := s.library.Facets(ctx)
	if err != nil {
		return nil, failure(err)
	}
	return &pb.ListFacetsResponse{Categories: result.Categories, Sources: result.Sources, Tags: result.Tags}, nil
}
