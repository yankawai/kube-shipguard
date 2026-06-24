package render

import (
	"reflect"
	"testing"
)

func TestHelmArgs(t *testing.T) {
	got := helmArgs(HelmOptions{
		Chart:     "charts/api",
		Release:   "api",
		Namespace: "prod",
		Values:    []string{"values.yaml", "values-prod.yaml"},
	})
	want := []string{
		"template", "api", "charts/api",
		"--namespace", "prod",
		"-f", "values.yaml",
		"-f", "values-prod.yaml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected helm args:\nwant %#v\ngot  %#v", want, got)
	}
}

func TestHelmArgsUsesDefaultRelease(t *testing.T) {
	got := helmArgs(HelmOptions{Chart: "charts/api"})
	want := []string{"template", "kube-shipguard", "charts/api"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected default helm args:\nwant %#v\ngot  %#v", want, got)
	}
}
