# nix-search

A small, fast CLI for [search.nixos.org](https://search.nixos.org). It talks to
the public Elasticsearch backend directly, using the same queries the website
issues, so results and ranking match the web UI without the JavaScript bundle or
a `nix search` evaluation.

```
pkgs          nixpkgs packages (default when no subcommand is given)
nixos         NixOS module options, e.g. services.nginx.enable
home-manager  Home Manager options, e.g. programs.git.enable
darwin        nix-darwin options, e.g. system.defaults.dock.autohide
service       options from service modules embedded in packages
```

## Usage

```console
$ nix-search ripgrep
ripgrep                15.2.0         rg
ripgrep-all            0.10.10        rga, rga-fzf, rga-preproc, rga-fzf-open
emacsPackages.ripgrep  20220520.1410

$ nix-search pkgs --programs gcc -n 3
$ nix-search nixos 'nginx enable' -o long
$ nix-search home-manager programs.git
$ nix-search -c 26.05 -s alpha -n 20 jq
$ nix-search -o json ffmpeg | jq
```

Global flags (`-c/--channel`, `-s/--sort`, `-n/--max-results`, `-o/--output`) may
appear before or after the subcommand, and values may be attached (`-n20`) or
separate (`-n 20`). The query does not need quoting when it is a single
argument, and multiple trailing words are joined with spaces.

Output styles are `regular` (default), `long` (adds descriptions and, for
options, the defining file), `short` (names only) and `json` (the raw documents
plus the total match count).

## Install

```console
$ nix run github:oscar3170/nix-search -- ripgrep
$ nix profile install github:oscar3170/nix-search
```

### As a flake input

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nix-search = {
      url = "github:oscar3170/nix-search";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, nix-search }: {
    # ...
    # packages.${system}.default        -> the nix-search binary
    # apps.${system}.default            -> `nix run`
    # overlays.default                  -> adds `nix-search` to pkgs
  };
}
```

For example, in a `devShell`:

```nix
packages = [
  nix-search.packages.${pkgs.stdenv.hostPlatform.system}.default
];
```

or with the overlay:

```nix
nixpkgs.overlays = [ nix-search.overlays.default ];
environment.systemPackages = [ pkgs.nix-search ];
```

## How it works

The backend serves one index per channel (`latest-<schema>-nixos-<channel>`)
holding packages, options, Home Manager options, nix-darwin options and service
options, separated by document type. `nix-search` sends the same `_search`
bodies as the website, including its rescoring of short attribute names, so the
result order matches what you would see in the browser.

The mapping schema version is compiled in, but the site bumps it whenever the
mapping changes. A request against a stale version gets a 404, in which case the
client asks the backend which aliases exist and retries with the newest one for
the requested channel. That also produces a useful error — with the list of
available channels — when the channel itself no longer exists.

Set `NIX_SEARCH_SCHEMA_VERSION` to skip straight to a specific mapping version
(used by the integration tests to exercise the fallback path).

## Development

```console
$ nix develop          # or use your own go 1.26 toolchain
$ go test -short ./... # unit tests, no network
$ go test ./...       # unit tests + live integration tests against search.nixos.org
```

The unit tests pin the Elasticsearch request bodies and the rendered output
against fixtures in `internal/search/testdata`, generated once from the original
Python implementation of this tool; `testdata/README.md` records the details.

`integration/` builds the CLI and runs it against the real backend, covering
each documentation set, JSON output, alias re-resolution and the unknown-channel
error. Those tests skip under `-short`, which is what the Nix build uses because
its sandbox has no network.
