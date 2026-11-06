// Command nix-search searches nixpkgs packages and NixOS, Home Manager and
// nix-darwin options through the public search.nixos.org backend.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/oscar3170/nix-search/internal/cli"
	"github.com/oscar3170/nix-search/internal/search"
	"github.com/oscar3170/nix-search/internal/version"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-V") {
		fmt.Printf("nix-search %s\n", version.Version)
		return
	}

	// Let an interrupt cancel the in-flight request instead of killing the
	// process mid-response.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(cli.MainContext(ctx, os.Args[1:], os.Stdout, os.Stderr, search.NewClient()))
}
