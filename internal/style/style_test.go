package style

import (
	"bytes"
	"testing"
)

func TestRender(t *testing.T) {
	plain := Palette{}
	if got := plain.Render(Bold|Cyan, "x"); got != "x" {
		t.Errorf("disabled palette rendered %q", got)
	}

	colored := Palette{Enabled: true}
	if got, want := colored.Render(Bold|Cyan, "x"), "\x1b[1m\x1b[36mx\x1b[0m"; got != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
	// Empty text and no attributes must not emit stray escape sequences.
	for _, tc := range []struct {
		attr Attr
		s    string
	}{{Bold, ""}, {0, "x"}} {
		if got := colored.Render(tc.attr, tc.s); got != tc.s {
			t.Errorf("Render(%v, %q) = %q, want the input", tc.attr, tc.s, got)
		}
	}
}

// TestEnabled pins the precedence between the flag and the environment, using a
// buffer (never a terminal) as the writer.
func TestEnabled(t *testing.T) {
	var out bytes.Buffer

	cases := []struct {
		name string
		mode Mode
		env  map[string]string
		want bool
	}{
		{"never", ModeNever, nil, false},
		{"always wins over NO_COLOR", ModeAlways, map[string]string{"NO_COLOR": "1"}, true},
		{"auto without a terminal", ModeAuto, nil, false},
		{"auto with NO_COLOR", ModeAuto, map[string]string{"NO_COLOR": "1"}, false},
		{"auto with CLICOLOR_FORCE", ModeAuto, map[string]string{"CLICOLOR_FORCE": "1"}, true},
		{"auto with CLICOLOR_FORCE=0", ModeAuto, map[string]string{"CLICOLOR_FORCE": "0"}, false},
		{"auto with a dumb terminal", ModeAuto, map[string]string{"TERM": "dumb"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			if got := Enabled(tc.mode, &out); got != tc.want {
				t.Errorf("Enabled(%q) = %v, want %v", tc.mode, got, tc.want)
			}
		})
	}
}
