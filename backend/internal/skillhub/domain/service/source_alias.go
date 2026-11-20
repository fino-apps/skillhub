package service

import (
	"errors"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
)

func sourceAliasesBySkill(aliases []repo.SourceAlias) map[string][]string {
	result := make(map[string][]string)
	for _, alias := range aliases {
		result[alias.SkillID] = append(result[alias.SkillID], alias.SourceKey)
	}
	return result
}

func rememberSource(tx repo.LibraryTx, skillID, key string, allowExisting bool) error {
	if key == "" {
		return nil
	}
	owner, err := tx.SkillBySource(key)
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return tx.CreateSourceAlias(repo.SourceAlias{SkillID: skillID, SourceKey: key})
	case err != nil:
		return err
	case !allowExisting && string(owner.ID) != skillID:
		return ErrConflict
	default:
		return nil
	}
}
