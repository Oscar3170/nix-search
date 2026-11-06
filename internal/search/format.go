package search

import (
	"encoding/json"
	"html"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/oscar3170/nix-search/internal/style"
)

// OutputFormat is one of the CLI's output styles.
type OutputFormat string

const (
	// OutputShort prints names only.
	OutputShort OutputFormat = "short"
	// OutputRegular prints one padded line per hit.
	OutputRegular OutputFormat = "regular"
	// OutputLong adds descriptions (and, for options, the defining file).
	OutputLong OutputFormat = "long"
	// OutputJSON prints the raw hits as JSON.
	OutputJSON OutputFormat = "json"
)

// ParseOutput resolves a --output value.
func ParseOutput(s string) (OutputFormat, bool) {
	switch OutputFormat(s) {
	case OutputShort, OutputRegular, OutputLong, OutputJSON:
		return OutputFormat(s), true
	}
	return "", false
}

// maxDefaultWidth is the length beyond which a default value is truncated in
// the non-long output styles.
const maxDefaultWidth = 60

// Write renders the response to w. JSON output is never colorized.
func Write(w io.Writer, resp *Response, source Source, format OutputFormat, palette style.Palette) error {
	var b strings.Builder

	switch {
	case format == OutputJSON:
		if err := writeJSON(&b, resp); err != nil {
			return err
		}
	case len(resp.Hits) == 0:
		b.WriteString("No results found.\n")
	case format == OutputShort:
		key := func(d Document) string { return d.OptionName }
		if source == SourcePkgs {
			key = func(d Document) string { return d.PackageAttrName }
		}
		for _, h := range resp.Hits {
			b.WriteString(palette.Render(style.Bold, key(h.Doc)))
			b.WriteByte('\n')
		}
	case source != SourcePkgs:
		writeOptions(&b, resp.Hits, format, palette)
	default:
		writePackages(&b, resp.Hits, format, palette)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// jsonOutput is the shape of `--output json`.
type jsonOutput struct {
	Total int               `json:"total"`
	Hits  []json.RawMessage `json:"hits"`
}

func writeJSON(b *strings.Builder, resp *Response) error {
	out := jsonOutput{
		Total: resp.Total,
		Hits:  make([]json.RawMessage, 0, len(resp.Hits)),
	}
	for _, h := range resp.Hits {
		out.Hits = append(out.Hits, h.Raw)
	}
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false) // keep descriptions readable, like Python's json.dumps
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeOptions(b *strings.Builder, hits []Hit, format OutputFormat, palette style.Palette) {
	nameW, typeW := 0, 0
	for _, h := range hits {
		nameW = max(nameW, utf8.RuneCountInString(h.Doc.OptionName))
		typeW = max(typeW, utf8.RuneCountInString(h.Doc.OptionType))
	}
	pad := strings.Repeat(" ", nameW+typeW+4)

	for _, h := range hits {
		d := h.Doc
		line := palette.Render(style.Bold, padRight(d.OptionName, nameW)) +
			"  " + palette.Render(style.Magenta, padRight(d.OptionType, typeW))
		if def := plainText(d.OptionDefault); def != "" {
			if format != OutputLong && utf8.RuneCountInString(def) > maxDefaultWidth {
				def = truncate(def, maxDefaultWidth-3) + "..."
			}
			line += "  " + palette.Render(style.Yellow, def)
		}
		if format == OutputLong {
			if desc := plainText(d.OptionDescription); desc != "" {
				line += "\n" + pad + palette.Render(style.Dim, desc)
			}
			if d.OptionSource != "" {
				line += "\n" + pad + palette.Render(style.Cyan, d.OptionSource)
			}
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func writePackages(b *strings.Builder, hits []Hit, format OutputFormat, palette style.Palette) {
	nameW, versionW := 0, 0
	for _, h := range hits {
		nameW = max(nameW, utf8.RuneCountInString(h.Doc.PackageAttrName))
		versionW = max(versionW, utf8.RuneCountInString(h.Doc.PackagePversion))
	}

	for _, h := range hits {
		d := h.Doc
		line := palette.Render(style.Bold, padRight(d.PackageAttrName, nameW)) +
			"  " + palette.Render(style.Yellow, padRight(d.PackagePversion, versionW))
		if programs := formatPrograms(d); programs != "" {
			line += "  " + palette.Render(style.Green, programs)
		}
		if format == OutputLong {
			if desc := strings.TrimSpace(strings.ReplaceAll(d.PackageDescription, "\n", " ")); desc != "" {
				line += "\n" + strings.Repeat(" ", nameW+versionW+4) + palette.Render(style.Dim, desc)
			}
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

// formatPrograms lists the executables a package provides, with the main
// program first.
func formatPrograms(d Document) string {
	progs := d.PackagePrograms
	if main := d.PackageMainProgram; main != "" {
		for _, p := range progs {
			if p != main {
				continue
			}
			reordered := make([]string, 0, len(progs))
			reordered = append(reordered, main)
			for _, other := range progs {
				if other != main {
					reordered = append(reordered, other)
				}
			}
			progs = reordered
			break
		}
	}
	return strings.Join(progs, ", ")
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

// plainText turns an option default or description (rendered HTML) into a
// single line of text.
func plainText(s string) string {
	if s == "" {
		return ""
	}
	s = tagRE.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// padRight pads s to width runes, the way Python's str.ljust does.
func padRight(s string, width int) string {
	if n := utf8.RuneCountInString(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

// truncate keeps the first n runes of s.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
