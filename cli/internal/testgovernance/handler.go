package testgovernance

import (
	"encoding/json"
	"fmt"
	"io"

	"cloudcc-customization-expert-go/internal/jsonx"
)

func Handle(action string, args []string, stdout io.Writer, cwd string) error {
	switch action {
	case "init":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc init test-governance <projectPath> <projectCode>")
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
	case "advise":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc advise testing <projectPath> <@change.json|json>")
		}
		body, err := jsonx.ParseEncodedObject(args[1], "testing change")
		if err != nil {
			return err
		}
		var change ChangeRequest
		if err := remarshal(body, &change); err != nil {
			return err
		}
		recommendation, err := Advise(args[0], change)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(recommendation)
	case "decide":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc decide testing <projectPath> <@decision.json|json>")
		}
		body, err := jsonx.ParseEncodedObject(args[1], "testing decision")
		if err != nil {
			return err
		}
		var request DecisionRequest
		if err := remarshal(body, &request); err != nil {
			return err
		}
		result, err := Decide(args[0], request)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(result)
	case "record":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc record testing <projectPath> <@run.json|json>")
		}
		body, err := jsonx.ParseEncodedObject(args[1], "testing run")
		if err != nil {
			return err
		}
		var request RunRequest
		if err := remarshal(body, &request); err != nil {
			return err
		}
		result, err := RecordRun(args[0], request)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(result)
	default:
		return fmt.Errorf("unsupported testing governance action %s", action)
	}
}

func remarshal(value any, target any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
