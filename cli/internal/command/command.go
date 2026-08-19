package command

import (
	"fmt"
	"io"
	"os"

	"cloudcc-customization-expert-go/internal/docs"
	"cloudcc-customization-expert-go/internal/modules"
	"cloudcc-customization-expert-go/internal/msapi"
	"cloudcc-customization-expert-go/internal/openapi"
	"cloudcc-customization-expert-go/internal/provider"
	"cloudcc-customization-expert-go/internal/version"
)

func Run(args []string, stdout io.Writer, stderr io.Writer, cwd string) int {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, "cloudcc: cannot resolve current directory:", err)
			return 1
		}
	}
	if len(args) == 0 {
		return version.Help(stdout, stderr)
	}
	if len(args) == 1 {
		if err := version.Handle(args[0], nil, stdout, stderr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	action, resource := args[0], args[1]
	rest := args[2:]

	if version.IsVersionAction(action) && resource != "version" && !(action == "doctor" && (resource == "classes" || resource == "provider")) {
		if err := version.Handle(action, append([]string{resource}, rest...), stdout, stderr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	var err error
	switch {
	case action == "doc":
		err = docs.Print(resource, firstOr(rest, ""), stdout)
	case action == "doctor" && resource == "provider":
		projectPath := firstOr(rest, cwd)
		err = provider.WriteDoctor(projectPath, stdout)
	case resource == "msapi" || resource == "metadata":
		if err = provider.RequireMSAPI(firstOr(rest, cwd)); err == nil {
			err = msapi.Handle(action, resource, rest, stdout, cwd)
		}
	case msapi.IsMetadataDomainAction(action) && msapi.IsMetadataDomain(resource):
		err = msapi.Handle(action, "msapi", append([]string{resource}, rest...), stdout, cwd)
	case msapi.IsLowCodeShortcut(action, resource):
		err = handleLowCodeShortcut(action, resource, rest, stdout, stderr, cwd)
	case resource == "openapi":
		err = openapi.Handle(action, rest, stdout, stderr, cwd)
	default:
		err = modules.Handle(action, resource, rest, stdout, stderr, cwd)
	}
	if err != nil {
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "请查看帮助：cloudcc --help", err)
		fmt.Fprintln(stderr)
		return 1
	}
	return 0
}

func handleLowCodeShortcut(action string, resource string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	selection, err := provider.ResolveForArgs(args, cwd)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "cloudcc low-code provider: %s (%s; safety=%s)\n", selection.SelectedMode, selection.Reason, selection.SafetyLevel)
	if selection.SelectedMode == provider.ModeMSAPI {
		return msapi.HandleLowCodeShortcut(action, resource, args, stdout, cwd)
	}
	return modules.Handle(action, resource, args, stdout, stderr, cwd)
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}
