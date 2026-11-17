package app

import (
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) versionView(row model.SkillVersionSnapshot) *pb.VersionView {
	return &pb.VersionView{
		Id: string(row.ID), SkillId: row.SkillId, Name: row.Name, Notes: row.Notes,
		SourceRef: row.SourceRef, CommitSha: row.CommitSha, ContentDigest: row.ContentDigest,
		Available: s.library.Available(row), CreateTime: row.CreateTime,
		UpstreamCommit: row.UpstreamCommit, UpstreamRepository: row.UpstreamRepository, UpstreamPath: row.UpstreamPath,
	}
}
