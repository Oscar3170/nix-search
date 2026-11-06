package cli

import (
	"strings"
	"unicode/utf8"

	"github.com/oscar3170/nix-search/internal/style"
)

// helpRow is one term/description pair of a help section.
type helpRow struct {
	// term is the subcommand or flag, rendered as syntax. Long-only flags are
	// written with the leading spaces that line them up under a short flag.
	term string
	// metavar is the value placeholder, rendered separately from the flag.
	metavar string
	// text is the untransformed description.
	text string
}

func (r helpRow) width() int {
	if r.metavar == "" {
		return utf8.RuneCountInString(r.term)
	}
	return utf8.RuneCountInString(r.term) + 1 + utf8.RuneCountInString(r.metavar)
}

var subcommandRows = []helpRow{
	{"pkgs", "", "nixpkgs packages (used when no subcommand is given)"},
	{"nixos", "", "NixOS module options, e.g. services.nginx.enable"},
	{"home-manager", "", "Home Manager options, e.g. programs.git.enable"},
	{"darwin", "", "nix-darwin options, e.g. system.defaults.dock.autohide"},
	{"service", "", "options from service modules embedded in packages"},
}

var pkgsFlagRows = []helpRow{
	{"--name", "", "match attribute/package names only"},
	{"--programs", "", "match provided program/executable names"},
}

var globalFlagRows = []helpRow{
	{"-c, --channel", "CHANNEL", "nixpkgs channel: unstable, 26.05, ... (default: unstable)"},
	{"-s, --sort", "MODE", "sort results: relevance, alpha (default: relevance)"},
	{"-n, --max-results", "N", "maximum number of results (default: 50)"},
	{"-o, --output", "FORMAT", "output style: short, regular, long, json (default: regular)"},
	{"    --color", "MODE", "colorize help: auto, always, never (default: auto)"},
	{"-h, --help", "", "show this help"},
	{"    --version", "", "show the version"},
}

// example is one line of the Examples section.
type example struct {
	args string
	// comment explains the flags; the leading "# " is added when rendering.
	comment string
}

var examples = []example{
	{"git", "packages (pkgs is the default)"},
	{"pkgs --name firefox", "match package names"},
	{"pkgs --programs gcc", "find packages shipping 'gcc'"},
	{"nixos 'nginx enable'", "NixOS options"},
	{"home-manager programs.git", "Home Manager options"},
	{"-c 26.05 -s alpha -n 20 jq", "channel 26.05, sorted by name"},
	{"-o json ffmpeg | jq", "machine-readable output"},
}

// progName is the program name used by the help text and examples.
const progName = "nix-search"

// Help renders the help text with p. Strip the palette to get the plain text.
func Help(p style.Palette) string {
	prog := p.Render(style.Bold|style.Cyan, progName)
	metavar := func(s string) string { return p.Render(style.Yellow, s) }

	var b strings.Builder
	b.WriteString(prog)
	b.WriteString(" - search nixpkgs packages and NixOS/Home Manager/darwin\n")
	b.WriteString("options via search.nixos.org.\n\n")

	b.WriteString(p.Render(style.Bold, "Usage:") + "\n")
	b.WriteString("  " + prog + " " + metavar("[flags]") + " " + metavar("<query>") + "\n")
	b.WriteString("  " + prog + " " + metavar("<subcommand>") + " " +
		metavar("[flags]") + " " + metavar("<query>") + "\n")

	writeSection(&b, p, "Subcommands", subcommandRows)
	writeSection(&b, p, "pkgs flags", pkgsFlagRows)
	writeSection(&b, p, "Global flags", globalFlagRows)

	b.WriteString("\nGlobal flags may appear before or after the subcommand, and values may be\n")
	b.WriteString("attached (-n20) or separate (-n 20).\n")

	writeExamples(&b, p)
	return b.String()
}

// writeSection renders a titled definition list, aligning the descriptions.
func writeSection(b *strings.Builder, p style.Palette, title string, rows []helpRow) {
	width := 0
	for _, row := range rows {
		width = max(width, row.width())
	}

	b.WriteString("\n" + p.Render(style.Bold, title+":") + "\n")
	for _, row := range rows {
		term := p.Render(style.Bold|style.Cyan, row.term)
		if row.metavar != "" {
			term += " " + p.Render(style.Yellow, row.metavar)
		}
		b.WriteString("  " + term + strings.Repeat(" ", width-row.width()+2) + row.text + "\n")
	}
}

// writeExamples renders the Examples section, aligning the comments.
func writeExamples(b *strings.Builder, p style.Palette) {
	const progpad = len(progName) + 1

	width := 0
	for _, ex := range examples {
		width = max(width, progpad+utf8.RuneCountInString(ex.args))
	}

	b.WriteString("\n" + p.Render(style.Bold, "Examples:") + "\n")
	for _, ex := range examples {
		cmd := p.Render(style.Bold|style.Cyan, progName) + " " + ex.args
		pad := strings.Repeat(" ", width-progpad-utf8.RuneCountInString(ex.args)+2)
		b.WriteString("  " + cmd + pad + p.Render(style.Dim, "# "+ex.comment) + "\n")
	}
}
