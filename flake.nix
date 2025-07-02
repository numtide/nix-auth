{
  description = "CLI tool to manage access tokens for Nix";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          default = self.packages.${system}.nix-auth;
          
          nix-auth = pkgs.buildGoModule {
            pname = "nix-auth";
            version = "0.1.0";
            
            src = self;
            
            vendorHash = "sha256-6NPryjjfK/YHbKwEIjgrVcaaik5QplAgyCgzec0OXFw=";
            
            meta = with pkgs.lib; {
              description = "CLI tool to manage access tokens for Nix";
              homepage = "https://github.com/numtide/nix-auth";
              license = licenses.mit;
              maintainers = with maintainers; [ numtide ];
            };
          };
        };
        
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            go-tools
            golangci-lint
            gopls
          ];
        };
      });
}
