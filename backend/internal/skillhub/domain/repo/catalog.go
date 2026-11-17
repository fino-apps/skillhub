package repo

import (
	"context"
	"errors"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

var ErrConflict = errors.New("catalog changed; refresh and retry")

type CatalogEntry struct {
	Skill   *model.Skill
	Version *model.SkillVersion
}

type CatalogQuery struct {
	Keyword, Category, Source string
	Tags                      []string
	Uncategorized             bool
	Offset, Limit             int
}

type Metadata struct {
	Category, Purpose, Notes, UpstreamName, UpstreamURL *string
	ReplaceTags, AddTags, RemoveTags                    []string
	Revision                                            string
	SkillID                                             model.SkillID
}

type CatalogRepo interface {
	AddSkill(context.Context, CatalogEntry) (bool, error)
	AddVersion(context.Context, *model.SkillVersion, bool) (*model.SkillVersion, error)
	List(context.Context, CatalogQuery) ([]*model.Skill, error)
	Tags(context.Context, []model.SkillID) (map[model.SkillID][]string, error)
	DefaultVersions(context.Context, []*model.Skill) (map[model.SkillID]*model.SkillVersion, error)
	Facets(context.Context) (categories, sources, tags []string, err error)
	UpdateMetadata(context.Context, []Metadata) error
	SetDefaultVersion(context.Context, *model.SkillVersion) error
	UpdateSnapshot(context.Context, *model.SkillVersion, string, string) error
	DeleteVersion(context.Context, *model.SkillVersion) error
	DeleteSkill(context.Context, model.SkillID) error
}
