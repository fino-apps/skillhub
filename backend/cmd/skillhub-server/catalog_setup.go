package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/fino-io/finokit/db"
	"gorm.io/gorm"
)

func prepareCatalog(ctx context.Context, provider db.Provider) error {
	return provider.NewSession(ctx).Transaction(func(tx *gorm.DB) error {
		var prior []struct {
			ID, LocalPath, RepositoryURL, RepositoryPath string
		}
		if err := tx.Table("skill").Where("source_key = ''").Find(&prior).Error; err != nil {
			return err
		}
		for _, row := range prior {
			kind, value, owner := "folder", row.LocalPath, ""
			if row.RepositoryURL != "" {
				kind, value = "github", row.RepositoryURL+"\x00"+row.RepositoryPath
				owner = strings.Split(strings.TrimPrefix(row.RepositoryURL, "https://github.com/"), "/")[0]
			}
			if value == "" {
				kind, value = "zip", row.ID
			}
			key := fmt.Sprintf("%x", sha256.Sum256([]byte(kind+"\x00"+value)))
			if err := tx.Exec(`UPDATE skill SET source_key = ?, import_kind = ?, upstream_name = ?, upstream_url = ?,
				default_version_id = COALESCE((SELECT id FROM skill_version WHERE skill_id = skill.id
				ORDER BY create_time DESC, id DESC LIMIT 1), '') WHERE id = ?`, key, kind, owner, row.RepositoryURL, row.ID).Error; err != nil {
				return err
			}
		}
		for _, statement := range []string{
			`UPDATE skill_version SET description = COALESCE((SELECT description FROM skill WHERE default_version_id = skill_version.id), '')
				WHERE description = '' AND EXISTS (SELECT 1 FROM skill WHERE default_version_id = skill_version.id)`,
			"DROP INDEX IF EXISTS idx_skill_location",
			"DROP INDEX IF EXISTS idx_skill_version_source",
			"CREATE UNIQUE INDEX IF NOT EXISTS skill_source_key ON skill(source_key)",
			"CREATE UNIQUE INDEX IF NOT EXISTS skill_version_commit ON skill_version(skill_id, commit_sha) WHERE commit_sha <> ''",
			"CREATE UNIQUE INDEX IF NOT EXISTS skill_version_digest ON skill_version(skill_id, content_digest) WHERE commit_sha = ''",
		} {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
