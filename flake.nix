{
  description = "nix-search: fast CLI for search.nixos.org (nixpkgs packages and NixOS/Home Manager/darwin options)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (pkgs:
        let
          nix-search = pkgs.buildGoModule {
            pname = "nix-search";
            inherit version;

            src = ./.;
            vendorHash = null;

            # The integration tests talk to search.nixos.org, which the build
            # sandbox cannot reach; -short leaves them out and keeps the unit
            # tests (which use committed fixtures) in.
            checkFlags = [ "-short" ];

            ldflags = [
              "-s"
              "-w"
              "-X github.com/oscar3170/nix-search/internal/version.Version=${version}"
            ];

            meta = {
              description = "Search nixpkgs packages and NixOS, Home Manager and nix-darwin options";
              homepage = "https://github.com/oscar3170/nix-search";
              mainProgram = "nix-search";
              platforms = systems;
            };
          };
        in
        {
          inherit nix-search;
          default = nix-search;
        });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/nix-search";
          meta.description = "Search nixpkgs packages and NixOS options";
        };
      });

      checks = forAllSystems (pkgs: {
        inherit (self.packages.${pkgs.stdenv.hostPlatform.system}) nix-search;
      });

      # Lets consumers pull the tool in with `overlays.default`.
      overlays.default = final: _prev: {
        nix-search = self.packages.${final.stdenv.hostPlatform.system}.default;
      };

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            delve
          ];
        };
      });
    };
}
