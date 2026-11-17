package repo

import (
	"context"
	"errors"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

var (
	ErrNotFound  = errors.New("library entry not found")
	ErrPageToken = errors.New("invalid pagination token")
)

type SkillFilter struct {
	Keyword, Category, Source string
	Tags                      []string
	Uncategorized             bool
}

type Facets struct {
	Categories, Sources, Tags []string
}

type Library interface {
	Transaction(context.Context, func(LibraryTx) error) error
	Skills(context.Context, SkillFilter, int, string) ([]model.SkillSnapshot, string, error)
	Versions(context.Context, string, int, string) ([]model.SkillVersionSnapshot, string, error)
	Skill(context.Context, string) (model.SkillSnapshot, error)
	Version(context.Context, string, string) (model.SkillVersionSnapshot, error)
	Tags(context.Context, []string) (map[string][]string, error)
	DefaultVersions(context.Context, []string) ([]model.SkillVersionSnapshot, error)
	Facets(context.Context) (Facets, error)
}

type LibraryTx interface {
	Skill(id string) (model.SkillSnapshot, error)
	SkillBySource(key string) (model.SkillSnapshot, error)
	Version(skill, id string) (model.SkillVersionSnapshot, error)
	MatchingVersion(skill, commit, digest string) (model.SkillVersionSnapshot, error)
	CreateSkill(model.SkillSnapshot) error
	CreateVersion(model.SkillVersionSnapshot) error
	UpdateSkill(model.SkillSnapshot, ...string) error
	ChangeTags(skill string, clear bool, add, remove []string) error
	DeleteVersion(model.SkillVersionSnapshot) error
	DeleteSkill(model.SkillSnapshot) ([]string, error)
}
