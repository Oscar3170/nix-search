package cli

import (
	"regexp"
	"strings"
	"testing"

	"github.com/oscar3170/nix-search/internal/search"
	"github.com/oscar3170/nix-search/internal/style"
)

// defaultOptions is the parsed form of a bare `nix-search <query>`.
func defaultOptions(text string) *Options {
	return &Options{
		Text:       text,
		Channel:    DefaultChannel,
		Source:     search.SourcePkgs,
		Mode:       search.ModeFullText,
		Sort:       search.SortRelevance,
		Output:     search.OutputRegular,
		MaxResults: DefaultMaxResults,
		Color:      style.ModeAuto,
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want *Options
	}{
		{
			name: "bare query means pkgs full-text",
			argv: []string{"git"},
			want: defaultOptions("git"),
		},
		{
			name: "multi-word query does not need quoting",
			argv: []string{"rfc", "allowed", "port"},
			want: defaultOptions("rfc allowed port"),
		},
		{
			name: "name mode",
			argv: []string{"pkgs", "--name", "firefox-eww"},
			want: func() *Options {
				o := defaultOptions("firefox-eww")
				o.Mode = search.ModeName
				return o
			}(),
		},
		{
			name: "programs flag before the subcommand",
			argv: []string{"--programs", "gcc"},
			want: func() *Options {
				o := defaultOptions("gcc")
				o.Mode = search.ModePrograms
				return o
			}(),
		},
		{
			name: "options source selects option mode",
			argv: []string{"nixos", "nginx", "enable"},
			want: func() *Options {
				o := defaultOptions("nginx enable")
				o.Source = search.SourceNixOS
				o.Mode = search.ModeOptions
				return o
			}(),
		},
		{
			name: "global flags before the subcommand",
			argv: []string{"-c", "26.05", "-s", "alpha", "-n", "20", "-o", "json",
				"home-manager", "programs.git"},
			want: func() *Options {
				o := defaultOptions("programs.git")
				o.Channel = "26.05"
				o.Sort = search.SortAlpha
				o.MaxResults = 20
				o.Output = search.OutputJSON
				o.Source = search.SourceHomeManager
				o.Mode = search.ModeOptions
				return o
			}(),
		},
		{
			name: "attached values and flags after the subcommand",
			argv: []string{"darwin", "-n3", "--sort=alpha", "--output=short", "dock"},
			want: func() *Options {
				o := defaultOptions("dock")
				o.Sort = search.SortAlpha
				o.MaxResults = 3
				o.Output = search.OutputShort
				o.Source = search.SourceDarwin
				o.Mode = search.ModeOptions
				return o
			}(),
		},
		{
			name: "color mode",
			argv: []string{"--color=always", "git"},
			want: func() *Options {
				o := defaultOptions("git")
				o.Color = style.ModeAlways
				return o
			}(),
		},
		{
			name: "double dash ends flag parsing",
			argv: []string{"--", "-foo"},
			want: defaultOptions("-foo"),
		},
		{
			name: "help short-circuits the query requirement",
			argv: []string{"-h"},
			want: func() *Options {
				o := defaultOptions("")
				o.Help = true
				return o
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.argv)
			if err != nil {
				t.Fatalf("Parse(%v): %v", tc.argv, err)
			}
			if *got != *tc.want {
				t.Errorf("Parse(%v) = %+v, want %+v", tc.argv, *got, *tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]struct {
		argv []string
		want string
	}{
		"no query":                {[]string{"pkgs"}, "missing search query"},
		"subcommand only":         {[]string{"nixos"}, "missing search query"},
		"unknown flag":            {[]string{"--bogus", "git"}, `unknown flag "--bogus"`},
		"missing flag value":      {[]string{"-n"}, "requires a value"},
		"non-numeric max-results": {[]string{"-n", "x", "git"}, "invalid max-results"},
		"zero max-results":        {[]string{"-n", "0", "git"}, "invalid max-results"},
		"unknown sort":            {[]string{"-s", "newest", "git"}, "invalid sort"},
		"unknown output":          {[]string{"-o", "yaml", "git"}, "invalid output"},
		"name on options source":  {[]string{"nixos", "--name", "foo"}, "only valid for the pkgs subcommand"},
		"name with a value":       {[]string{"--name=1", "foo"}, "does not take a value"},
		"malformed channel":       {[]string{"-c", "bad channel", "git"}, "invalid channel"},
		"unknown color mode":      {[]string{"--color", "sometimes", "git"}, "invalid color"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(tc.argv)
			if err == nil {
				t.Fatalf("Parse(%v) succeeded, want an error", tc.argv)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestHelpColorIsOnlyStyling checks that colorizing the help never changes the
// text a terminal-less user sees.
func TestHelpColorIsOnlyStyling(t *testing.T) {
	plain := Help(style.Palette{})
	coloredHelp := Help(style.Palette{Enabled: true})

	if !strings.Contains(coloredHelp, "\x1b[") {
		t.Error("colored help has no escape sequences")
	}
	if got := ansiRE.ReplaceAllString(coloredHelp, ""); got != plain {
		t.Errorf("help differs from plain after stripping ANSI\nwant:\n%q\ngot:\n%q", plain, got)
	}
	for _, want := range []string{
		"Usage:", "Subcommands:", "pkgs flags:", "Global flags:", "Examples:",
		"home-manager", "--programs", "--color MODE",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("help does not mention %q", want)
		}
	}
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")
