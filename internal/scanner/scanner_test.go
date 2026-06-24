package scanner

import "testing"

func TestLoadYAMLParsesRenderedStream(t *testing.T) {
	resources, err := LoadYAML("helm:api", []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
---
apiVersion: v1
kind: Service
metadata:
  name: api
`))
	if err != nil {
		t.Fatalf("load rendered yaml: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected two resources, got %#v", resources)
	}
	if resources[0].File != "helm:api" {
		t.Fatalf("expected rendered source name, got %s", resources[0].File)
	}
}
