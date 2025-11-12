{
  description = "Go bindings for the Tree-sitter parsing library";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    self.submodules = true;
  };

  outputs =
    inputs:
    let
      inherit (inputs.nixpkgs) lib;
      inherit (inputs) self;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      eachSystem = lib.genAttrs systems;
      pkgsFor = inputs.nixpkgs.legacyPackages;
    in
    {
      packages = eachSystem (
        system:
        let
          pkgs = pkgsFor.${system};
          inherit (pkgs) lib;
        in
        {
          default = pkgs.buildGoModule {
            pname = "go-tree-sitter";
            version = "0.25.1";

            src = self;

            vendorHash = "sha256-6rj6oNohxBQt0LhIaHh3fQKHbNCsLsBkuPYNquHEVzE=";
            proxyVendor = true;

            subPackages = [ "." ];

            meta = {
              description = "Go bindings for Tree-sitter parsing library";
              homepage = "https://github.com/tree-sitter/go-tree-sitter";
              license = lib.licenses.mit;
              maintainers = [ lib.maintainers.amaanq ];
            };
          };
        }
      );

      devShells = eachSystem (
        system:
        let
          pkgs = pkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = [
              pkgs.go
              pkgs.gopls
            ];
          };
        }
      );

      checks = eachSystem (system: {
        inherit (self.packages.${system}) default;
      });
    };
}
