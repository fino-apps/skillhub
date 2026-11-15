package service

import (
	"fmt"
	"regexp"
)

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// InstallCommand generates a command to run in the target project.
func InstallCommand(repository, path, commit, name string, global bool) (string, error) {
	repository, err := NormalizeRepository(repository)
	if err != nil {
		return "", err
	}
	path, err = repositoryPath(path)
	if err != nil {
		return "", err
	}
	if !commitPattern.MatchString(commit) || len(name) > 64 || !validSkillName.MatchString(name) {
		return "", fmt.Errorf("invalid version or skill name")
	}
	source := repository + "#" + commit
	if path != "" {
		source = repository + "/tree/" + commit + "/" + path
	}
	command := "npx skills@1.8.1 add '" + source + "' --skill '" + name + "' -a codex --copy"
	if global {
		command += " -g"
	}
	return command, nil
}
