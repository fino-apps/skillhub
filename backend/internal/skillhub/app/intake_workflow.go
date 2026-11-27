package app

import (
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/internal/skillhub/safecount"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func candidateViews(items []service.Candidate) []*pb.SkillCandidate {
	result := make([]*pb.SkillCandidate, 0, len(items))
	for _, item := range items {
		result = append(result, &pb.SkillCandidate{Path: item.Path, Name: item.Name, Description: item.Description})
	}
	return result
}

func importResponse(result *service.ImportResult) *pb.ImportSkillResponse {
	response := &pb.ImportSkillResponse{AddedCount: safecount.Int32(int64(result.Added)), ExistingCount: safecount.Int32(int64(result.Existing))}
	for _, item := range result.Failures {
		response.Failures = append(response.Failures, &pb.ImportFailure{Path: item.Path, Message: failure(item.Err).Error()})
	}
	return response
}
