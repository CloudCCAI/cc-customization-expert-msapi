package jsonx

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

func ParseEncodedObject(encoded string, label string) (map[string]any, error) {
	if encoded == "" {
		return nil, fmt.Errorf("%s: encodedBodyJson is required", label)
	}
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(encoded, "@") {
		return ReadObjectFile(strings.TrimPrefix(encoded, "@"))
	}
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s: encodedBodyJson is not valid URI-encoded JSON: %w", label, err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(decoded), &body); err != nil {
		return nil, fmt.Errorf("%s: encodedBodyJson is not valid JSON: %w", label, err)
	}
	if body == nil {
		return nil, fmt.Errorf("%s: request body must be a JSON object", label)
	}
	return body, nil
}

func ReadObjectFile(file string) (map[string]any, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func WriteObjectFile(file string, data map[string]any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(file, b, 0644)
}
