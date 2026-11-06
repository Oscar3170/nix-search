package cli

// Usage is the help text, printed for -h/--help and when no arguments are
// given.
const Usage = `nix-search - search nixpkgs packages and NixOS/Home Manager/darwin
options via search.nixos.org.

Usage:
  nix-search [flags] <query>
  nix-search <subcommand> [flags] <query>

Subcommands:
  pkgs          nixpkgs packages (used when no subcommand is given)
  nixos         NixOS module options, e.g. services.nginx.enable
  home-manager  Home Manager options, e.g. programs.git.enable
  darwin        nix-darwin options, e.g. system.defaults.dock.autohide
  service       options from service modules embedded in packages

pkgs flags:
  --name        match attribute/package names only
  --programs    match provided program/executable names

Global flags:
  -c, --channel CHANNEL  nixpkgs channel: unstable, 26.05, ... (default: unstable)
  -s, --sort MODE        sort results: relevance, alpha (default: relevance)
  -n, --max-results N    maximum number of results (default: 50)
  -o, --output FORMAT    output style: short, regular, long, json (default: regular)
  -h, --help             show this help
      --version          show the version

Global flags may appear before or after the subcommand, and values may be
attached (-n20) or separate (-n 20).

Examples:
  nix-search git                        # packages (pkgs is the default)
  nix-search pkgs --name firefox        # match package names
  nix-search pkgs --programs gcc        # find packages shipping 'gcc'
  nix-search nixos 'nginx enable'       # NixOS options
  nix-search home-manager programs.git  # Home Manager options
  nix-search -c 26.05 -s alpha -n 20 jq # channel 26.05, sorted by name
  nix-search -o json ffmpeg | jq        # machine-readable output
`
