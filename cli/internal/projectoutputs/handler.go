package projectoutputs

import (
	"encoding/json"
	"fmt"
	"io"
)

func Handle(action string, args []string, stdout io.Writer, cwd string) error {
	switch action {
	case "init":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc init project-outputs <projectPath> <projectCode>")
		}
		result, err := Init(args[0], args[1])
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(result)
	case "doctor":
		projectPath := cwd
		if len(args) > 0 {
			projectPath = args[0]
		}
		return WriteDoctor(projectPath, stdout)
	default:
		return fmt.Errorf("unsupported project outputs action %s", action)
	}
}
