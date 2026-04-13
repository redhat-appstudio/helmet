package framework

import (
	"strings"
	"testing"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/subcmd"
)

func TestStandardIntegrationsOrderMatchesSubcmd(t *testing.T) {
	t.Parallel()
	std := StandardIntegrations()
	internal := subcmd.StandardModules()
	if len(std) != len(internal) {
		t.Fatalf("length mismatch: StandardIntegrations=%d StandardModules=%d", len(std), len(internal))
	}
	for i := range std {
		if std[i].Name != internal[i].Name {
			t.Fatalf("position %d: got name %q want %q", i, std[i].Name, internal[i].Name)
		}
	}
}

func TestSelectIntegrations_emptyNamesReturnsSameSlice(t *testing.T) {
	t.Parallel()
	mods := []api.IntegrationModule{{Name: "a"}, {Name: "b"}}
	out, err := SelectIntegrations(mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Name != "a" || out[1].Name != "b" {
		t.Fatalf("unexpected result: %#v", out)
	}
	if &out[0] != &mods[0] {
		t.Fatal("expected same slice backing array when names is empty")
	}
}

func TestSelectIntegrations_nilSliceEmptyNamesReturnsNil(t *testing.T) {
	t.Parallel()
	out, err := SelectIntegrations(nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatalf("expected nil slice when modules is nil and names is empty, got %#v", out)
	}
}

func TestSelectIntegrations_subsetOrder(t *testing.T) {
	t.Parallel()
	mods := StandardIntegrations()
	out, err := SelectIntegrations(mods, "quay", "github")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	if out[0].Name != "quay" || out[1].Name != "github" {
		t.Fatalf("order: got %q, %q", out[0].Name, out[1].Name)
	}
}

func TestSelectIntegrations_dedupPreservesFirstOrder(t *testing.T) {
	t.Parallel()
	mods := StandardIntegrations()
	out, err := SelectIntegrations(mods, "github", "quay", "github", "quay")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Name != "github" || out[1].Name != "quay" {
		t.Fatalf("got %#v", namesOf(out))
	}
}

func TestSelectIntegrations_unknownName(t *testing.T) {
	t.Parallel()
	mods := StandardIntegrations()
	_, err := SelectIntegrations(mods, "nope")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error should mention unknown name: %v", err)
	}
}

func TestSelectIntegrations_multipleUnknownSortedInMessage(t *testing.T) {
	t.Parallel()
	mods := StandardIntegrations()
	_, err := SelectIntegrations(mods, "zebra", "apple")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "apple") || !strings.Contains(msg, "zebra") {
		t.Fatalf("expected both names in error: %v", err)
	}
	// Sorted for stable diagnostics: "apple" before "zebra"
	if strings.Index(msg, "apple") > strings.Index(msg, "zebra") {
		t.Fatalf("expected sorted unknown names in message: %v", err)
	}
}

func TestSelectIntegrations_customModuleList(t *testing.T) {
	t.Parallel()
	custom := []api.IntegrationModule{
		{Name: "one"},
		{Name: "two"},
	}
	out, err := SelectIntegrations(custom, "two", "one")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Name != "two" || out[1].Name != "one" {
		t.Fatalf("got %#v", namesOf(out))
	}
}

func namesOf(mods []api.IntegrationModule) []string {
	s := make([]string, len(mods))
	for i, m := range mods {
		s[i] = m.Name
	}
	return s
}
