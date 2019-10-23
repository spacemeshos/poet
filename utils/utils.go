package utils

import (
	"os/exec"
	"strings"
)

// GetPkgPath gets a repo name and returns the folder in which
// the installed repo resides
func GetPkgPath(repo string) (string, error) {
	mockSrcCodeDir, err := exec.Command("go", "list", "-m", "-f", "'{{.Dir}}'", repo).Output()
	if err != nil {
		return "", err
	}
	// string returned with open "'" and closing "'"
	trimmedStr := strings.Replace(string(mockSrcCodeDir), "'", "", -1)
	// remove redundant whitespaces
	trimmedStr = strings.TrimSpace(trimmedStr)

	return trimmedStr, nil
}
