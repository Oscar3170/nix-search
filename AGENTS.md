# Agent Instructions

## Commands

| Task | Command |
|------|---------|
| Unit tests (offline) | `go test ./...` |
| Live backend tests | `NIX_SEARCH_INTEGRATION=1 go test ./integration` |
| Format | `gofmt -w .` |
| Vet | `go vet ./...` |
| Run locally | `go run ./cmd/nix-search -n 5 ripgrep` |
| Build (runs tests) | `nix build .#` |
| Dev shell | `nix develop` |

## Key Conventions

- `internal/search/testdata/` pins parity with the original Python implementation: `query/` is the exact Elasticsearch request body, `output/` the exact rendered text. Never hand-edit goldens; they are the spec.
- Ranking is defined by the request bodies, not by result order, so do not "simplify" `query.go` clauses or boost values.
- Color is applied per cell after column padding; JSON output is never colorized. `TestColorKeepsLayout` requires stripping ANSI from colored output to reproduce the plain output byte for byte.
- Live tests hit a shared public backend (search.nixos.org): keep them gated behind `NIX_SEARCH_INTEGRATION=1` and keep `minInterval` spacing in `integration/integration_test.go`.
- The flake builds with `-short` because its sandbox has no network; keep new tests offline-safe by default.
- `NIX_SEARCH_SCHEMA_VERSION` exists to force the alias-discovery path; it is not needed in normal use.
- Adding a source subcommand means touching `internal/search/source.go`, `query.go`, `internal/cli/help.go` and adding request-body goldens.

## External References

| Need | File |
|------|------|
| Usage, flags, install | `README.md` |
| Testdata provenance | `internal/search/testdata/README.md` |
| Color palette and previews | `examples/README.md` |
| Package output and checks | `flake.nix` |
