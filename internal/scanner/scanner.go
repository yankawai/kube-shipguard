package scanner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Resource struct {
	File       string
	APIVersion string
	Kind       string
	Metadata   Metadata
	Spec       map[string]any
	Data       map[string]any
	StringData map[string]any
}

type Metadata struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

func Load(paths []string) ([]Resource, error) {
	if len(paths) == 0 {
		return nil, errors.New("at least one path is required")
	}

	var resources []Resource
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}

		if stat.IsDir() {
			err = filepath.WalkDir(path, func(filePath string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !isYAML(filePath) {
					return nil
				}
				loaded, err := loadFile(filePath)
				if err != nil {
					return err
				}
				resources = append(resources, loaded...)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}

		if !isYAML(path) {
			continue
		}
		loaded, err := loadFile(path)
		if err != nil {
			return nil, err
		}
		resources = append(resources, loaded...)
	}

	return resources, nil
}

func loadFile(path string) ([]Resource, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadYAML(path, content)
}

func LoadYAML(source string, content []byte) ([]Resource, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var resources []Resource
	for {
		var doc map[string]any
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", source, err)
		}
		if len(doc) == 0 {
			continue
		}

		resource := Resource{
			File:       source,
			APIVersion: stringValue(doc["apiVersion"]),
			Kind:       stringValue(doc["kind"]),
			Metadata:   metadataValue(doc["metadata"]),
			Spec:       mapValue(doc["spec"]),
			Data:       mapValue(doc["data"]),
			StringData: mapValue(doc["stringData"]),
		}
		if resource.Kind == "" {
			continue
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func isYAML(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".yaml" || extension == ".yml"
}

func metadataValue(value any) Metadata {
	metadata := mapValue(value)
	return Metadata{
		Name:      stringValue(metadata["name"]),
		Namespace: defaultString(stringValue(metadata["namespace"]), "default"),
		Labels:    stringMap(mapValue(metadata["labels"])),
	}
}

func mapValue(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return typed
}

func stringMap(value map[string]any) map[string]string {
	result := make(map[string]string, len(value))
	for key, raw := range value {
		if text := stringValue(raw); text != "" {
			result[key] = text
		}
	}
	return result
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
