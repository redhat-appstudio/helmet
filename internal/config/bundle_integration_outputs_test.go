package config

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"gopkg.in/yaml.v3"
)

func TestDiscoverProductIntegrationSecrets_fromBundleTemplates(t *testing.T) {
	fs := fstest.MapFS{
		"bundles/tpa/charts/tssc-tpa/templates/integration-secret.yaml": &fstest.MapFile{
			Data: []byte(`---
apiVersion: v1
kind: Secret
metadata:
  name: tssc-trustification-integration
stringData:
  bombastic_api_url: "x"
  supported_cyclonedx_version: "y"
`),
		},
	}
	cfs := chartfs.New(fs)
	got, err := discoverProductIntegrationSecrets(cfs, "tpa", "tssc-tpa")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bombastic_api_url", "supported_cyclonedx_version"}
	gotKeys := got["tssc-trustification-integration"]
	if len(gotKeys) != len(want) {
		t.Fatalf("keys %v want %v", gotKeys, want)
	}
	for i := range want {
		if gotKeys[i] != want[i] {
			t.Fatalf("keys[%d]=%q want %q", i, gotKeys[i], want[i])
		}
	}
}

func TestValidateProductIntegrationOutputs_typeEnforced(t *testing.T) {
	fs := fstest.MapFS{}
	cfs := chartfs.New(fs)
	err := ValidateProductIntegrationOutputs(cfs, "demo", "tssc-demo", []BundleOutput{{
		Name: "wrong",
		Type: "secret",
	}})
	if err == nil {
		t.Fatal("expected error for invalid type/name")
	}
}

func TestProductMarshalYAML_omitsOutputs(t *testing.T) {
	p := Product{
		Name:       "Demo Product",
		Properties: map[string]interface{}{"k": true},
		Outputs: []BundleOutput{{
			Name: "tssc-demo-integration",
			Type: OutputTypeIntegrationSecret,
			Data: []string{"secret"},
		}},
	}
	b, err := yaml.Marshal(&p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "outputs") {
		t.Fatalf("cluster-facing YAML must omit outputs:\n%s", b)
	}
}
