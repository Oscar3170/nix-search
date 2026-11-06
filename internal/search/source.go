package search

// Source identifies one documentation set served by the search backend. Every
// set lives in the same Elasticsearch index and is separated by document type.
type Source string

const (
	// SourcePkgs is the nixpkgs package set.
	SourcePkgs Source = "pkgs"
	// SourceNixOS is the NixOS module option set.
	SourceNixOS Source = "nixos"
	// SourceHomeManager is the Home Manager option set.
	SourceHomeManager Source = "home-manager"
	// SourceDarwin is the nix-darwin option set.
	SourceDarwin Source = "darwin"
	// SourceService is the set of options from service modules embedded in
	// packages.
	SourceService Source = "service"
)

// Sources lists every source in the order used by the help output.
var Sources = []Source{
	SourcePkgs,
	SourceNixOS,
	SourceHomeManager,
	SourceDarwin,
	SourceService,
}

// docTypes maps a source to the Elasticsearch `type` of the documents it
// searches.
var docTypes = map[Source]string{
	SourcePkgs:        "package",
	SourceNixOS:       "option",
	SourceHomeManager: "home-manager-option",
	SourceDarwin:      "darwin-option",
	SourceService:     "service",
}

// ParseSource resolves a subcommand name to a Source.
func ParseSource(s string) (Source, bool) {
	src := Source(s)
	_, ok := docTypes[src]
	return src, ok
}

// DocType is the Elasticsearch document type searched by the source.
func (s Source) DocType() string { return docTypes[s] }

// Mode is the kind of match to perform.
type Mode int

const (
	// ModeFullText searches names, programs and descriptions. It is the
	// default for packages.
	ModeFullText Mode = iota
	// ModeName matches attribute and package names only.
	ModeName
	// ModePrograms matches packages by the executables they provide.
	ModePrograms
	// ModeOptions searches module options. Every non-package source uses it.
	ModeOptions
)

// String names the mode.
func (m Mode) String() string {
	switch m {
	case ModeFullText:
		return "full-text"
	case ModeName:
		return "name"
	case ModePrograms:
		return "programs"
	case ModeOptions:
		return "options"
	}
	return "unknown"
}

// Sort orders the results.
type Sort string

const (
	// SortRelevance is the backend's relevance order (the default).
	SortRelevance Sort = "relevance"
	// SortAlpha is alphabetical order by attribute/option name.
	SortAlpha Sort = "alpha"
)

// ParseSort resolves a --sort value.
func ParseSort(s string) (Sort, bool) {
	switch Sort(s) {
	case SortRelevance:
		return SortRelevance, true
	case SortAlpha:
		return SortAlpha, true
	}
	return "", false
}

// Query is one search request.
type Query struct {
	// Text is the search term.
	Text string
	// Channel is the nixpkgs channel, e.g. "unstable" or "26.05".
	Channel string
	// Source is the documentation set to search.
	Source Source
	// Mode is the kind of match to perform.
	Mode Mode
	// Sort orders the results.
	Sort Sort
	// Size is the maximum number of hits to return.
	Size int
}
