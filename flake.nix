{
  description = "Terraform Provider for RunPod";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { pkgs, system, ... }: 
      let
        # Allow unfree packages (terraform uses BSL license)
        pkgsUnfree = import inputs.nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
      in {
        devShells.default = pkgsUnfree.mkShell {
          buildInputs = with pkgsUnfree; [
            go
            terraform
            golangci-lint
            goreleaser
            gnupg
          ];

          shellHook = ''
            echo "Terraform Provider for RunPod - Development Shell"
            echo "Go version: $(go version)"
            echo "Terraform version: $(terraform version | head -1)"
          '';
        };
      };
    };
}
