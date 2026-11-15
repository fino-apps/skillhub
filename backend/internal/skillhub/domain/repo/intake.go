package repo

import (
	"context"
	"errors"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
)

var ErrSkillExists = errors.New("skill already exists")

// IntakeRepo commits a Skill and its initial version together.
type IntakeRepo interface {
	CreateInitial(ctx context.Context, skill *model.Skill, version *model.SkillVersion, publish func() error) error
}
