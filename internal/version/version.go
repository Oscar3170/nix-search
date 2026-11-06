// Package version holds the build-time version string.
package version

// Version is overridden at build time with
//
//	-ldflags "-X github.com/oscar3170/nix-search/internal/version.Version=v1.2.3"
var Version = "dev"
