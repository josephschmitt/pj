{
  description = "pj - Project Finder CLI Tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "pj";
          version = self.rev or "dev";
          src = ./.;
          vendorHash = "sha256-rya2afSV9Y1hmUZU0wyR9NETBl3TXD/OTHv0zvVl8v8=";

          ldflags = [
            "-X main.version=${self.rev or "dev"}"
          ];

          meta = with pkgs.lib; {
            description = "Fast Go CLI tool for discovering project directories";
            homepage = "https://github.com/josephschmitt/pj";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            gnumake
            git
            lefthook
          ];

          shellHook = ''
            echo "pj development environment"
            echo "Go version: $(go version)"
            lefthook install
          '';
        };
      }
    );
}
