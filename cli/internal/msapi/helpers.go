package msapi

import "strings"

func validationRuleApplyOptions(args []string) (bool, string) {
	execute := false
	approval := ""
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--execute":
			execute = true
		case arg == "--dry-run":
			execute = false
		case arg == "--approval" && i+1 < len(args):
			approval = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--approval="):
			approval = strings.TrimSpace(strings.TrimPrefix(arg, "--approval="))
		}
	}
	return execute, approval
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
