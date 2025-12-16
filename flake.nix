{
  description = "CLI tool to manage access tokens for Nix";

  nixConfig = {
    extra-substituters = [ "https://cache.numtide.com" ];
    extra-trusted-public-keys = [ "cache.numtide.com-1:GF3TabtFocLtonIGfz3PD61AgIO8GmjCYhEAmYy4VPY=" ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    treefmt-nix.url = "github:numtide/treefmt-nix";
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          self',
          ...
        }:
        {
          packages = {
            default = self'.packages.nix-auth;

            nix-auth = pkgs.buildGoModule {
              pname = "nix-auth";
              version = "0.1.0";

              src = self;

              vendorHash = pkgs.lib.fileContents ./nix/vendorHash.txt;

              meta = with pkgs.lib; {
                description = "CLI tool to manage access tokens for Nix";
                homepage = "https://github.com/numtide/nix-auth";
                license = licenses.mit;
                maintainers = with maintainers; [ numtide ];
              };
            };
          };

          treefmt = {
            projectRootFile = "flake.nix";
            programs = {
              nixfmt.enable = true;
              gofumpt.enable = true;
            };
          };

          checks = {
            build = self'.packages.nix-auth;

            go-test = self'.packages.nix-auth.overrideAttrs (old: {
              name = "go-test";
              buildPhase = ''
                HOME=$TMPDIR go test -v ./...
              '';
              doCheck = false;
              installPhase = ''
                touch $out
              '';
              fixupPhase = ":";
            });

            golangci-lint = self'.packages.nix-auth.overrideAttrs (old: {
              name = "golangci-lint";
              nativeBuildInputs = old.nativeBuildInputs ++ [ pkgs.golangci-lint ];
              buildPhase = ''
                HOME=$TMPDIR golangci-lint run
              '';
              doCheck = false;
              installPhase = ''
                touch $out
              '';
              fixupPhase = ":";
            });
          };

          devShells.default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              go-tools
              golangci-lint
              gopls
              goreleaser
            ];

            inputsFrom = [ config.treefmt.build.devShell ];
          };
        };
    };
}
