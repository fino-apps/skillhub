package app

import (
	"strings"

	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

// validSkillLocation requires exactly one comparison source.
func validSkillLocation(location *pb.SkillLocation) bool {
	if location == nil {
		return false
	}
	count := 0
	for _, value := range []string{location.GetCurrentSkillId(), location.GetVersionId(), location.GetLocalPath()} {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}
	if location.GetGithub() != nil {
		count++
	}
	return count == 1
}
