package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestBodyMatchesReference compares the generated request bodies with the ones
// the original Python implementation produced (testdata/query). Ranking on
// search.nixos.org depends on these exact queries, so any drift is a bug.
func TestBodyMatchesReference(t *testing.T) {
	cases := map[string]Query{
		"pkgs_fulltext_relevance": {
			Text: "ripgrep", Channel: "unstable", Source: SourcePkgs,
			Mode: ModeFullText, Sort: SortRelevance, Size: 50,
		},
		"pkgs_fulltext_alpha": {
			Text: "ripgrep", Channel: "unstable", Source: SourcePkgs,
			Mode: ModeFullText, Sort: SortAlpha, Size: 20,
		},
		"pkgs_fulltext_rfc_allowed_port": {
			Text: "rfc allowed port", Channel: "unstable", Source: SourcePkgs,
			Mode: ModeFullText, Sort: SortRelevance, Size: 5,
		},
		"pkgs_name_relevance": {
			Text: "firefox-eww", Channel: "unstable", Source: SourcePkgs,
			Mode: ModeName, Sort: SortRelevance, Size: 50,
		},
		"pkgs_programs_relevance": {
			Text: "gcc", Channel: "unstable", Source: SourcePkgs,
			Mode: ModePrograms, Sort: SortRelevance, Size: 50,
		},
		"nixos_options_relevance": {
			Text: "nginx enable", Channel: "unstable", Source: SourceNixOS,
			Mode: ModeOptions, Sort: SortRelevance, Size: 50,
		},
		"nixos_options_alpha": {
			Text: "nginx enable", Channel: "unstable", Source: SourceNixOS,
			Mode: ModeOptions, Sort: SortAlpha, Size: 10,
		},
		"nixos_options_repeated_words": {
			Text: "foo foo", Channel: "unstable", Source: SourceNixOS,
			Mode: ModeOptions, Sort: SortRelevance, Size: 50,
		},
		"home_manager_options_relevance": {
			Text: "programs.git", Channel: "unstable", Source: SourceHomeManager,
			Mode: ModeOptions, Sort: SortRelevance, Size: 50,
		},
		"darwin_options_relevance": {
			Text: "dock autohide", Channel: "unstable", Source: SourceDarwin,
			Mode: ModeOptions, Sort: SortRelevance, Size: 25,
		},
		"service_options_relevance": {
			Text: "autoconnect", Channel: "unstable", Source: SourceService,
			Mode: ModeOptions, Sort: SortRelevance, Size: 50,
		},
		"service_options_hyphen_underscore": {
			Text: "systemd_services-foo bar", Channel: "unstable", Source: SourceService,
			Mode: ModeOptions, Sort: SortRelevance, Size: 50,
		},
	}

	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := json.Marshal(query.body())
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			want := readTestdata(t, filepath.Join("query", name+".json"))
			compareJSON(t, want, got)
		})
	}
}

// compareJSON compares two JSON documents structurally, so key order (which
// Elasticsearch ignores) does not matter.
func compareJSON(t *testing.T, want, got []byte) {
	t.Helper()

	var wantValue, gotValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal expected JSON: %v", err)
	}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal actual JSON: %v", err)
	}
	if reflect.DeepEqual(wantValue, gotValue) {
		return
	}

	wantPretty, _ := json.MarshalIndent(wantValue, "", "  ")
	gotPretty, _ := json.MarshalIndent(gotValue, "", "  ")
	t.Errorf("body mismatch\nwant:\n%s\n\ngot:\n%s", wantPretty, gotPretty)
}

func readTestdata(t *testing.T, parts ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, parts...)...))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	return data
}
