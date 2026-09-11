package modules

import (
	"fmt"
	"strings"
)

type javaToken struct {
	text  string
	line  int
	depth int
}

type javaTypeDeclaration struct {
	kind      string
	name      string
	line      int
	depth     int
	modifiers map[string]bool
}

func javaSourceStructurePolicy(source string, expectedClass string) ([]string, []string) {
	declarations := javaNamedTypeDeclarations(source)
	var violations []string
	var warnings []string
	var topLevel []javaTypeDeclaration
	for _, declaration := range declarations {
		if declaration.depth == 0 {
			topLevel = append(topLevel, declaration)
			continue
		}
		if declaration.depth == 1 && declaration.modifiers["private"] && declaration.modifiers["static"] {
			warnings = append(warnings, fmt.Sprintf("nested %s %s at line %d should remain a small implementation detail; prefer private methods or a separate CloudCC custom class", declaration.kind, declaration.name, declaration.line))
			continue
		}
		violations = append(violations, fmt.Sprintf("named nested/local %s %s at line %d is not allowed; use private methods or a private static nested type only for a small data holder", declaration.kind, declaration.name, declaration.line))
	}

	if len(topLevel) == 0 {
		violations = append(violations, fmt.Sprintf("source must declare one top-level public class named %s", expectedClass))
		return violations, warnings
	}
	main := topLevel[0]
	if main.kind != "class" || main.name != expectedClass || !main.modifiers["public"] {
		violations = append(violations, fmt.Sprintf("top-level type at line %d must be public class %s", main.line, expectedClass))
	}
	for _, declaration := range topLevel[1:] {
		violations = append(violations, fmt.Sprintf("additional top-level %s %s at line %d is not allowed; create a separate CloudCC custom class resource", declaration.kind, declaration.name, declaration.line))
	}
	return violations, warnings
}

func javaFragmentTypePolicyViolations(source string, resource string) []string {
	declarations := javaNamedTypeDeclarations(source)
	violations := make([]string, 0, len(declarations))
	for _, declaration := range declarations {
		violations = append(violations, fmt.Sprintf("%s source must remain a thin executable fragment; named %s %s at line %d is not allowed, so move cohesive logic into methods or a separate CloudCC custom class", resource, declaration.kind, declaration.name, declaration.line))
	}
	return violations
}

func javaOptionalResourceClassPolicy(source string, expectedClass string) ([]string, []string) {
	if len(javaNamedTypeDeclarations(source)) == 0 {
		return nil, nil
	}
	return javaSourceStructurePolicy(source, expectedClass)
}

func javaNamedTypeDeclarations(source string) []javaTypeDeclaration {
	tokens := lexJavaStructureTokens(source)
	declarations := []javaTypeDeclaration{}
	for i, token := range tokens {
		if token.text != "class" && token.text != "interface" && token.text != "enum" && token.text != "record" {
			continue
		}
		if i > 0 && tokens[i-1].text == "." {
			continue
		}
		nameIndex := i + 1
		if nameIndex >= len(tokens) || !isJavaIdentifierToken(tokens[nameIndex].text) {
			continue
		}
		modifiers := map[string]bool{}
		for j := i - 1; j >= 0; j-- {
			previous := tokens[j]
			if previous.depth != token.depth || previous.text == ";" || previous.text == "{" || previous.text == "}" {
				break
			}
			switch previous.text {
			case "public", "protected", "private", "abstract", "static", "final", "strictfp", "sealed", "non-sealed":
				modifiers[previous.text] = true
			}
		}
		declarations = append(declarations, javaTypeDeclaration{
			kind:      token.text,
			name:      tokens[nameIndex].text,
			line:      token.line,
			depth:     token.depth,
			modifiers: modifiers,
		})
	}
	return declarations
}

func lexJavaStructureTokens(source string) []javaToken {
	tokens := []javaToken{}
	line := 1
	depth := 0
	for i := 0; i < len(source); {
		switch {
		case source[i] == '\n':
			line++
			i++
		case source[i] == '\r' || source[i] == ' ' || source[i] == '\t' || source[i] == '\f':
			i++
		case i+1 < len(source) && source[i:i+2] == "//":
			i += 2
			for i < len(source) && source[i] != '\n' {
				i++
			}
		case i+1 < len(source) && source[i:i+2] == "/*":
			i += 2
			for i < len(source) {
				if i+1 < len(source) && source[i:i+2] == "*/" {
					i += 2
					break
				}
				if source[i] == '\n' {
					line++
				}
				i++
			}
		case i+2 < len(source) && source[i:i+3] == `"""`:
			i += 3
			for i < len(source) {
				if i+2 < len(source) && source[i:i+3] == `"""` {
					i += 3
					break
				}
				if source[i] == '\n' {
					line++
				}
				i++
			}
		case source[i] == '"' || source[i] == '\'':
			quote := source[i]
			i++
			for i < len(source) {
				if source[i] == '\n' {
					line++
				}
				if source[i] == '\\' && i+1 < len(source) {
					i += 2
					continue
				}
				if source[i] == quote {
					i++
					break
				}
				i++
			}
		case isJavaIdentifierByte(source[i]):
			start := i
			i++
			for i < len(source) && isJavaIdentifierByte(source[i]) {
				i++
			}
			tokens = append(tokens, javaToken{text: source[start:i], line: line, depth: depth})
		default:
			symbol := string(source[i])
			if symbol == "}" && depth > 0 {
				depth--
			}
			if strings.Contains("{}.;@()[]=,<>?", symbol) {
				tokens = append(tokens, javaToken{text: symbol, line: line, depth: depth})
			}
			if symbol == "{" {
				depth++
			}
			i++
		}
	}
	return tokens
}

func isJavaIdentifierByte(value byte) bool {
	return value == '_' || value == '$' || value >= 0x80 || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func isJavaIdentifierToken(value string) bool {
	if value == "" || strings.Contains("{}.;@()[]=,<>?", value) {
		return false
	}
	first := value[0]
	return first == '_' || first == '$' || first >= 0x80 || first >= 'a' && first <= 'z' || first >= 'A' && first <= 'Z'
}
