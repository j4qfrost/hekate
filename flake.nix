{
  description = "hekate — venue-first event coordination on the AT Protocol (Go server + CLI, SvelteKit web)";

  # Pinned to a tarball URL (not `github:`) so the flake resolves without
  # hitting api.github.com — matters behind corporate proxies / shared-IP CI.
  # nixos-unstable is required here: server/go.mod declares `go 1.26`, which
  # only the unstable channel's Go (1.26.x) satisfies.
  inputs.nixpkgs.url = "https://github.com/NixOS/nixpkgs/archive/refs/heads/nixos-unstable.tar.gz";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
    in {
      packages = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };

          # server/ is the v0.1 reference component. It pulls indigo + pgx +
          # OpenTelemetry, so it needs a real vendorHash (fixed-output derivation
          # that fetches the module graph once). The hash below was materialised
          # by setting it to lib.fakeHash, building, and copying the reported hash.
          server = pkgs.buildGoModule {
            pname = "hekate-server";
            version = "0.1.0";
            src = ./server;
            vendorHash = "sha256-Irrp0aZfSXMrzdE6lo4Tv+39WL3mC0g8Wx3sFSuip/c=";
            # Only build the server binary; skip building every internal pkg main.
            subPackages = [ "cmd/hekate-server" ];
            # Integration tests need Postgres; keep the nix check build-only.
            doCheck = false;
            ldflags = [ "-s" "-w" ];
            meta = {
              description = "hekate indexer / read API";
              mainProgram = "hekate-server";
            };
          };

          # cli/ is stdlib-only (no go.sum, no external requires), so the module
          # graph is empty — vendorHash = null is correct here.
          cli = pkgs.buildGoModule {
            pname = "hekate-cli";
            version = "0.1.0";
            src = ./cli;
            vendorHash = null;
            subPackages = [ "cmd/hekate" ];
            doCheck = true;
            ldflags = [ "-s" "-w" ];
            meta = {
              description = "hekate reference CLI client";
              mainProgram = "hekate";
            };
          };
        in {
          inherit server cli;
          default = server;
        });

      # `nix flake check` builds these. Both are pure buildGoModule derivations
      # (deps resolved via fixed-output vendorHash) so they work in the sandbox.
      # We deliberately do NOT add a Postgres-backed integration-test check.
      checks = forAllSystems (system: {
        server = self.packages.${system}.server;
        cli = self.packages.${system}.cli;
      });

      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.mkShell {
            buildInputs = [
              # Go server + CLI
              pkgs.go
              pkgs.gopls
              pkgs.gotools
              # sqlc / goose codegen + migrations for server/
              pkgs.sqlc
              pkgs.goose
              # web/ (SvelteKit) — built via pnpm inside this shell only; there
              # is no pure pnpm2nix package output for web/ (see README note).
              pkgs.nodejs_22
              pkgs.pnpm
              # Postgres + PostGIS client (psql) for deploy/ + lex-validate flows
              pkgs.postgresql
              pkgs.postgresql.pkgs.postgis
              # lex-validate is a Python meta-schema check (make lex-validate)
              pkgs.python312
              # deploy/ smoke stack
              pkgs.docker-compose
            ];

            shellHook = ''
              echo "hekate dev shell — go $(go version | awk '{print $3}'), node $(node --version), pnpm $(pnpm --version)"
              echo "  server: cd server && go test ./...   |   cli: cd cli && go test ./..."
              echo "  web:    cd web && pnpm install && pnpm build   (devShell-only — not packaged)"
            '';
          };
        });
    };
}
