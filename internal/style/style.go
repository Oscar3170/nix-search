// Package style renders ANSI styling for terminal output.
package style

import (
	"io"
	"os"
	"strings"
)

// Attr is a set of text attributes, combinable with |.
type Attr uint8

const (
	Bold Attr = 1 << iota
	Dim
	Green
	Yellow
	Magenta
	Cyan
)

// sgr lists the attributes in rendering order, so a combination always produces
// the same escape sequence.
var sgr = []struct {
	attr Attr
	code string
}{
	{Bold, "\x1b[1m"},
	{Dim, "\x1b[2m"},
	{Green, "\x1b[32m"},
	{Yellow, "\x1b[33m"},
	{Magenta, "\x1b[35m"},
	{Cyan, "\x1b[36m"},
}

const reset = "\x1b[0m"

// Palette renders text with ANSI attributes when Enabled is set, and returns it
// unchanged otherwise, so callers never branch on color support.
type Palette struct{ Enabled bool }

// Render wraps s in the escape sequences for attr.
func (p Palette) Render(attr Attr, s string) string {
	if !p.Enabled || s == "" || attr == 0 {
		return s
	}

	var b strings.Builder
	for _, a := range sgr {
		if attr&a.attr != 0 {
			b.WriteString(a.code)
		}
	}
	b.WriteString(s)
	b.WriteString(reset)
	return b.String()
}

// Mode is how the CLI decides whether to colorize.
type Mode string

const (
	// ModeAuto colors only when writing to a terminal.
	ModeAuto Mode = "auto"
	// ModeAlways colors even when redirected, for previews and logs.
	ModeAlways Mode = "always"
	// ModeNever disables color.
	ModeNever Mode = "never"
)

// ParseMode resolves a --color value.
func ParseMode(s string) (Mode, bool) {
	switch Mode(s) {
	case ModeAuto, ModeAlways, ModeNever:
		return Mode(s), true
	}
	return "", false
}

// Enabled reports whether output to w should be colorized under mode. The auto
// mode follows the NO_COLOR, CLICOLOR_FORCE and TERM=dumb conventions; an
// explicit always/never wins over all of them.
func Enabled(mode Mode, w io.Writer) bool {
	switch mode {
	case ModeAlways:
		return true
	case ModeNever:
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if v := os.Getenv("CLICOLOR_FORCE"); v != "" && v != "0" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}

// isTerminal reports whether w is a character device.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
