package app

import (
	"context"
	"errors"

	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func failure(err error) error {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return errcode.WrapPlatformNotFound(err)
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

func rpcError(ctx context.Context, operation string, err error) error {
	mapped := failure(err)
	var reason string
	switch {
	case errors.Is(err, repo.ErrPageToken):
		reason = "invalid_page_token"
	case errcode.IsPlatformBadRequest(mapped):
		reason = "invalid_input"
	case errcode.IsPlatformNotFound(mapped):
		reason = "not_found"
	case errcode.IsPlatformConflict(mapped):
		reason = "conflict"
	}
	if reason != "" {
		logs.Ctx(ctx).Warnw(operation+" rejected", "reason", reason)
	}
	return mapped
}
