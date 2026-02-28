package utils

import (
	"fmt"
	"strings"
)

func ValidateBranchPrefix(branch, requiredPrefix string) error {
	if !strings.HasPrefix(branch, requiredPrefix) {
		return fmt.Errorf("ветка должна начинаться с префикса '%s'", requiredPrefix)
	}
	return nil
}

func MakeCIBranchName(branch, prefix, ciPrefix string) string {
	suffix := strings.TrimPrefix(branch, prefix)
	return prefix + ciPrefix + suffix
}
