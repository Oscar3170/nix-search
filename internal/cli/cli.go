// Package cli implements the nix-search command line: flag parsing, help
// output and wiring of the search client.
package cli

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/oscar3170/nix-search/internal/search"
	"github.com/oscar3170/nix-search/internal/style"
)

// Defaults for the global flags.
const (
	DefaultChannel    = "unstable"
	DefaultMaxResults = 50
)

// channelRE keeps channel names from breaking the index URL.
var channelRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Options is a parsed command line.
type Options struct {
	// Text is the search term: the remaining positional arguments joined by a
	// space, so quoting is optional.
	Text       string
	Channel    string
	Source     search.Source
	Mode       search.Mode
	Sort       search.Sort
	Output     search.OutputFormat
	MaxResults int
	// Color decides whether the help text is colorized.
	Color style.Mode
	// Help reports that -h/--help was given.
	Help bool
}

// Request builds the search query described by the options.
func (o *Options) Request() search.Query {
	return search.Query{
		Text:    o.Text,
		Channel: o.Channel,
		Source:  o.Source,
		Mode:    o.Mode,
		Sort:    o.Sort,
		Size:    o.MaxResults,
	}
}

// paletteFor resolves the palette for the given output stream.
func paletteFor(w io.Writer, mode style.Mode) style.Palette {
	return style.Palette{Enabled: style.Enabled(mode, w)}
}

// Searcher is the part of search.Client the CLI needs.
type Searcher interface {
	Search(ctx context.Context, q search.Query) (*search.Response, error)
}

// Main runs the CLI and returns the process exit code. Flags are accepted
// before and after the subcommand, and the subcommand may be omitted for
// package searches.
func Main(argv []string, stdout, stderr io.Writer, searcher Searcher) int {
	return MainContext(context.Background(), argv, stdout, stderr, searcher)
}

// MainContext is Main with a caller-supplied context for the search request.
func MainContext(ctx context.Context, argv []string, stdout, stderr io.Writer, searcher Searcher) int {
	if len(argv) == 0 {
		fmt.Fprint(stderr, Help(paletteFor(stderr, style.ModeAuto)))
		return 2
	}

	opts, err := Parse(argv)
	if err != nil {
		fmt.Fprintf(stderr, "nix-search: error: %v\n", err)
		fmt.Fprintln(stderr, "Try 'nix-search --help' for more information.")
		return 2
	}
	if opts.Help {
		fmt.Fprint(stdout, Help(paletteFor(stdout, opts.Color)))
		return 0
	}

	resp, err := searcher.Search(ctx, opts.Request())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := search.Write(stdout, resp, opts.Source, opts.Output); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// Parse turns argv into Options.
func Parse(argv []string) (*Options, error) {
	opts := &Options{
		Channel:    DefaultChannel,
		Source:     search.SourcePkgs,
		Mode:       search.ModeFullText,
		Sort:       search.SortRelevance,
		Output:     search.OutputRegular,
		MaxResults: DefaultMaxResults,
		Color:      style.ModeAuto,
	}

	var words []string
	var match string // --name or --programs, without the dashes
	sourceSet := false
	positional := false

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		if positional || !isFlag(arg) {
			if !positional {
				// Only the first positional can name a subcommand, so a query
				// that happens to match a source name stays a query.
				if src, ok := search.ParseSource(arg); ok && !sourceSet && len(words) == 0 {
					opts.Source = src
					sourceSet = true
					continue
				}
			}
			words = append(words, arg)
			continue
		}

		if arg == "--" {
			positional = true
			continue
		}

		name, inline, hasInline := splitFlag(arg)
		// value returns the flag's value, either attached (-n20, --sort=alpha)
		// or taken from the next argument.
		value := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			if i+1 >= len(argv) {
				return "", fmt.Errorf("flag %s requires a value", name)
			}
			i++
			return argv[i], nil
		}

		switch name {
		case "-h", "--help":
			opts.Help = true
		case "-c", "--channel":
			v, err := value()
			if err != nil {
				return nil, err
			}
			if !channelRE.MatchString(v) {
				return nil, fmt.Errorf("invalid channel %q", v)
			}
			opts.Channel = v
		case "-s", "--sort":
			v, err := value()
			if err != nil {
				return nil, err
			}
			sortOrder, ok := search.ParseSort(v)
			if !ok {
				return nil, fmt.Errorf("invalid sort %q (want relevance or alpha)", v)
			}
			opts.Sort = sortOrder
		case "-n", "--max-results":
			v, err := value()
			if err != nil {
				return nil, err
			}
			n, convErr := strconv.Atoi(v)
			if convErr != nil || n <= 0 {
				return nil, fmt.Errorf("invalid max-results %q (want a positive integer)", v)
			}
			opts.MaxResults = n
		case "-o", "--output":
			v, err := value()
			if err != nil {
				return nil, err
			}
			format, ok := search.ParseOutput(v)
			if !ok {
				return nil, fmt.Errorf("invalid output %q (want short, regular, long or json)", v)
			}
			opts.Output = format
		case "--color":
			v, err := value()
			if err != nil {
				return nil, err
			}
			mode, ok := style.ParseMode(v)
			if !ok {
				return nil, fmt.Errorf("invalid color %q (want auto, always or never)", v)
			}
			opts.Color = mode
		case "--name", "--programs":
			if hasInline {
				return nil, fmt.Errorf("flag %s does not take a value", name)
			}
			match = strings.TrimPrefix(name, "--")
		default:
			return nil, fmt.Errorf("unknown flag %q", arg)
		}
	}

	if opts.Help {
		return opts, nil
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("missing search query")
	}
	opts.Text = strings.Join(words, " ")

	switch {
	case match != "" && opts.Source != search.SourcePkgs:
		return nil, fmt.Errorf("--%s is only valid for the pkgs subcommand", match)
	case match == "name":
		opts.Mode = search.ModeName
	case match == "programs":
		opts.Mode = search.ModePrograms
	case opts.Source != search.SourcePkgs:
		opts.Mode = search.ModeOptions
	}
	return opts, nil
}

// isFlag reports whether arg looks like a flag rather than a positional value.
func isFlag(arg string) bool {
	return len(arg) > 1 && arg[0] == '-'
}

// splitFlag splits a flag into its name and, when attached, its value.
// It understands `--sort=alpha`, `-n20` and `-n=20`.
func splitFlag(arg string) (name, value string, hasValue bool) {
	if strings.HasPrefix(arg, "--") {
		if n, v, ok := strings.Cut(arg, "="); ok {
			return n, v, true
		}
		return arg, "", false
	}
	if len(arg) > 2 {
		return arg[:2], strings.TrimPrefix(arg[2:], "="), true
	}
	return arg, "", false
}
