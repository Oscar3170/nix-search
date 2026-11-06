package search

import (
	"reflect"
	"testing"
)

func TestParseAliasCatalog(t *testing.T) {
	// Plain-text shape of `_cat/aliases?h=alias`, including aliases that must
	// be ignored.
	out := `latest-51-nixos-unstable
latest-52-nixos-unstable
latest-52-nixos-26.05
latest-50-nixos-unstable
latest-52-group-manual
latest-52-images
.kibana
not-an-alias

`
	catalog := parseAliasCatalog(out)

	want := indexCatalog{
		"unstable": {51, 52, 50},
		"26.05":    {52},
	}
	if !reflect.DeepEqual(catalog, want) {
		t.Errorf("catalog = %v, want %v", catalog, want)
	}

	if latest, ok := catalog.newest("unstable"); !ok || latest != 52 {
		t.Errorf("newest(unstable) = %d, %v; want 52, true", latest, ok)
	}
	if _, ok := catalog.newest("25.05"); ok {
		t.Error("newest(25.05) should not exist")
	}
	if got, want := catalog.channels(), []string{"26.05", "unstable"}; !reflect.DeepEqual(got, want) {
		t.Errorf("channels = %v, want %v", got, want)
	}
}

// TestParseAliasCatalogJSON covers the JSON body the backend returns when the
// request has `Accept: application/json`.
func TestParseAliasCatalogJSON(t *testing.T) {
	out := `[{"alias":"latest-52-nixos-unstable"},{"alias":"latest-52-nixos-26.05"},{"alias":".kibana"}]`
	catalog := parseAliasCatalog(out)
	if latest, ok := catalog.newest("unstable"); !ok || latest != 52 {
		t.Errorf("newest(unstable) = %d, %v; want 52, true", latest, ok)
	}
	if got, want := catalog.channels(), []string{"26.05", "unstable"}; !reflect.DeepEqual(got, want) {
		t.Errorf("channels = %v, want %v", got, want)
	}
}
