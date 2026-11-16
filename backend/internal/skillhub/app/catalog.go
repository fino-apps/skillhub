package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/skillhub/backend/internal/skillhub/app/mapper"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
	"github.com/google/uuid"
)

func (s skillhubServer) listSkills(ctx context.Context, in *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	size := int(in.GetPageSize())
	if size <= 0 || size > 100 {
		size = 40
	}
	scope := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%q/%q/%q/%q/%t",
		in.GetKeyword(), in.GetCategory(), in.GetSource(), in.GetTags(), in.GetUncategorized()))))
	offset := 0
	if token := in.GetPageToken(); token != "" {
		prefix, value, ok := strings.Cut(token, ".")
		var err error
		offset, err = strconv.Atoi(value)
		if !ok || prefix != scope || err != nil || offset < 0 {
			return nil, errcode.NewPlatformBadRequest()
		}
	}

	skills, err := s.catalog.List(ctx, repo.CatalogQuery{
		Keyword: strings.TrimSpace(in.GetKeyword()), Category: in.GetCategory(), Source: in.GetSource(),
		Tags: in.GetTags(), Uncategorized: in.GetUncategorized(), Offset: offset, Limit: size + 1,
	})
	if err != nil {
		return nil, catalogError(err)
	}
	var next string
	if len(skills) > size {
		skills = skills[:size]
		next = scope + "." + strconv.Itoa(offset+size)
	}
	views, err := s.skillViews(ctx, skills)
	if err != nil {
		return nil, catalogError(err)
	}
	return &pb.ListSkillsResponse{Skills: views, NextPageToken: next}, nil
}

func (s skillhubServer) getSkill(ctx context.Context, in *pb.GetSkillRequest) (*pb.GetSkillResponse, error) {
	skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId()))
	if err != nil {
		return nil, catalogError(err)
	}
	views, err := s.skillViews(ctx, []*model.Skill{skill})
	if err != nil {
		return nil, catalogError(err)
	}
	return &pb.GetSkillResponse{Skill: views[0]}, nil
}

func (s skillhubServer) skillViews(ctx context.Context, skills []*model.Skill) ([]*pb.SkillView, error) {
	ids := make([]model.SkillID, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ID())
	}
	tags, err := s.catalog.Tags(ctx, ids)
	if err != nil {
		return nil, err
	}
	versions, err := s.catalog.DefaultVersions(ctx, skills)
	if err != nil {
		return nil, err
	}
	views := mapper.ToSkillViews(skills)
	for i, id := range ids {
		views[i].Tags = tags[id]
		if version := versions[id]; version != nil {
			views[i].DefaultVersion = s.versionView(version)
		}
	}
	return views, nil
}

func (s skillhubServer) updateSkill(ctx context.Context, in *pb.UpdateSkillRequest) (*pb.UpdateSkillResponse, error) {
	value := in.GetSkill()
	if value == nil {
		return nil, errcode.NewPlatformBadRequest()
	}
	tags, err := normalizeTags(value.GetTags())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	link := strings.TrimSpace(value.GetUpstreamUrl())
	if link != "" && !validSourceURL(link) {
		return nil, errcode.WrapPlatformBadRequest(errors.New("上游链接必须是 HTTP 或 HTTPS"))
	}
	update := repo.Metadata{
		SkillID: model.SkillID(in.GetSkillId()), Category: clean(value.GetCategory()),
		Purpose: clean(value.GetPurpose()), Notes: clean(value.GetNotes()),
		UpstreamName: clean(value.GetUpstreamName()), UpstreamURL: &link, ReplaceTags: tags,
	}
	if err := s.catalog.UpdateMetadata(ctx, []repo.Metadata{update}); err != nil {
		return nil, catalogError(err)
	}
	result, err := s.getSkill(ctx, &pb.GetSkillRequest{SkillId: in.GetSkillId()})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSkillResponse{Skill: result.Skill}, nil
}

func (s skillhubServer) batchUpdateSkillMetadata(ctx context.Context, in *pb.BatchUpdateSkillMetadataRequest) (*core.Null, error) {
	updates := make([]repo.Metadata, 0, len(in.GetChanges()))
	for _, change := range in.GetChanges() {
		if uuid.Validate(change.GetSkillId()) != nil || utf8.RuneCountInString(change.GetCategory()) > 128 ||
			utf8.RuneCountInString(change.GetPurpose()) > 240 {
			return nil, errcode.NewPlatformBadRequest()
		}
		add, err := normalizeTags(change.GetAddTags())
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		remove, err := normalizeTags(change.GetRemoveTags())
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		update := repo.Metadata{SkillID: model.SkillID(change.GetSkillId()), Revision: change.GetRevision(), AddTags: add, RemoveTags: remove}
		if change.Category != nil {
			update.Category = clean(change.GetCategory())
		}
		if change.Purpose != nil {
			update.Purpose = clean(change.GetPurpose())
		}
		updates = append(updates, update)
	}
	if err := s.catalog.UpdateMetadata(ctx, updates); err != nil {
		return nil, catalogError(err)
	}
	return &core.Null{}, nil
}

func clean(value string) *string {
	value = strings.TrimSpace(value)
	return &value
}

func normalizeTags(items []string) ([]string, error) {
	if len(items) > 100 {
		return nil, errors.New("标签不能超过 100 项")
	}
	tags := make([]string, 0, len(items))
	for _, value := range items {
		tag := strings.TrimSpace(value)
		if tag == "" || utf8.RuneCountInString(tag) > 128 {
			return nil, errors.New("标签必须为 1 至 128 个字符")
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func validSourceURL(link string) bool {
	parsed, err := url.Parse(link)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") &&
		parsed.Hostname() != "" && parsed.User == nil
}

func catalogError(err error) error {
	switch {
	case errors.Is(err, repo.ErrConflict):
		return errcode.WrapPlatformConflict(err)
	case core.IsNotFoundError(err):
		return errcode.WrapPlatformNotFound(err)
	default:
		return errcode.WrapGetSkillhubFailed(err)
	}
}
