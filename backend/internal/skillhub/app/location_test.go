package app

import (
	"testing"

	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func TestValidSkillLocation(t *testing.T) {
	tests := []struct {
		name  string
		value *pb.SkillLocation
		want  bool
	}{
		{"missing", nil, false},
		{"empty", &pb.SkillLocation{}, false},
		{"personal", &pb.SkillLocation{CurrentSkillId: "id"}, true},
		{"github", &pb.SkillLocation{Github: &pb.GitHubSource{}}, true},
		{"multiple", &pb.SkillLocation{VersionId: "id", LocalPath: "/tmp/skill"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validSkillLocation(tt.value); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
