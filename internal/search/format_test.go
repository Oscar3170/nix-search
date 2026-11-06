package search

import (
	"strings"
	"testing"
)

// TestWriteMatchesReference renders committed backend responses and compares
// them with the output of the original Python implementation
// (testdata/output), which pins column widths, truncation and HTML stripping.
func TestWriteMatchesReference(t *testing.T) {
	cases := []struct {
		fixture string
		source  Source
		format  OutputFormat
	}{
		{"pkgs", SourcePkgs, OutputShort},
		{"pkgs", SourcePkgs, OutputRegular},
		{"pkgs", SourcePkgs, OutputLong},
		{"pkgs", SourcePkgs, OutputJSON},
		{"pkgs_edge", SourcePkgs, OutputShort},
		{"pkgs_edge", SourcePkgs, OutputRegular},
		{"pkgs_edge", SourcePkgs, OutputLong},
		{"pkgs_edge", SourcePkgs, OutputJSON},
		{"options", SourceNixOS, OutputShort},
		{"options", SourceNixOS, OutputRegular},
		{"options", SourceNixOS, OutputLong},
		{"options", SourceNixOS, OutputJSON},
		{"options_edge", SourceNixOS, OutputShort},
		{"options_edge", SourceNixOS, OutputRegular},
		{"options_edge", SourceNixOS, OutputLong},
		{"options_edge", SourceNixOS, OutputJSON},
		{"home_manager", SourceHomeManager, OutputRegular},
		{"home_manager", SourceHomeManager, OutputJSON},
		{"empty", SourcePkgs, OutputRegular},
		{"empty", SourcePkgs, OutputJSON},
	}

	for _, tc := range cases {
		name := tc.fixture + "/" + string(tc.format)
		t.Run(name, func(t *testing.T) {
			resp, err := parseResponse(readTestdata(t, "responses", tc.fixture+".json"))
			if err != nil {
				t.Fatalf("parse response: %v", err)
			}

			var out strings.Builder
			if err := Write(&out, resp, tc.source, tc.format); err != nil {
				t.Fatalf("write: %v", err)
			}

			want := string(readTestdata(t, "output", tc.fixture+"_"+string(tc.format)+".txt"))
			if out.String() != want {
				t.Errorf("output mismatch\nwant:\n%q\n\ngot:\n%q", want, out.String())
			}
		})
	}
}

func TestPlainText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"<rendered-html><p>Whether to enable &quot;foo&quot;.</p>\n</rendered-html>", `Whether to enable "foo".`},
		{"<p>a <b>b</b>\n  c</p>", "a b c"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := plainText(tc.in); got != tc.want {
			t.Errorf("plainText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTruncateKeepsRunes(t *testing.T) {
	// Python slices code points, so multibyte names must not be cut mid-rune.
	if got := truncate("áéíóú", 2); got != "áé" {
		t.Errorf("truncate = %q, want %q", got, "áé")
	}
	if got := truncate("abc", 5); got != "abc" {
		t.Errorf("truncate = %q, want %q", got, "abc")
	}
}
