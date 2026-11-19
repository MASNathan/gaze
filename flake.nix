{
  description = "Gaze source code outlines";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { nixpkgs, ... }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.buildGoModule.override { go = pkgs.go_1_27; } {
            pname = "gaze";
            version = "0.2.0";
            src = nixpkgs.lib.cleanSourceWith {
              src = ./.;
              filter = path: type: baseNameOf path != "vendor" && nixpkgs.lib.cleanSourceFilter path type;
            };
            proxyVendor = true;
            vendorHash = "sha256-ophPnxoFpAYpNSHHVenCbteMorvpEnFnTYVEY+YD1ZA=";
            subPackages = [ "cmd/gaze" ];
            env.CGO_ENABLED = 1;
            meta = {
              description = "Compact source code outlines";
              homepage = "https://github.com/masnathan/gaze";
              license = nixpkgs.lib.licenses.mit;
              mainProgram = "gaze";
            };
          };
        });
    };
}
