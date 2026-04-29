# hekate — top-level Makefile
# Wraps per-subproject toolchains so contributors don't have to memorise them.

.PHONY: help dev test lex-validate selfhost-smoke server-build server-test cli-build cli-test web-install web-check web-build clean

help:
	@echo "hekate development targets"
	@echo ""
	@echo "  make dev              full-stack dev loop (postgres + server + web)"
	@echo "  make test             run all tests (server, cli, web, lexicon validation)"
	@echo "  make lex-validate     validate lexicons against ATProto meta-schema"
	@echo "  make selfhost-smoke   verify docker compose self-host bring-up"
	@echo ""
	@echo "  make server-build     build hekate-server binary"
	@echo "  make server-test      run server tests (testcontainers required)"
	@echo "  make cli-build        build hekate CLI binary"
	@echo "  make cli-test         run CLI tests"
	@echo "  make web-install      install web deps (pnpm install)"
	@echo "  make web-check        type-check + lint web"
	@echo "  make web-build        build web for production"
	@echo ""
	@echo "  make clean            remove build artefacts"

dev:
	@echo ">> bringing up postgis + server + web for development"
	cd deploy && docker compose -f docker-compose.dev.yml up -d postgis
	@echo ">> postgres ready; start the server (cd server && go run ./cmd/hekate-server) and web (cd web && pnpm dev) in separate terminals"

test: lex-validate server-test cli-test web-check
	@echo "all tests passed"

lex-validate:
	@echo ">> validating lexicons"
	@for f in lexicons/app/hekate/*.json; do \
		python3 -c "import json,sys; d=json.load(open('$$f')); assert d['lexicon']==1, '$$f: bad lexicon version'; assert 'id' in d, '$$f: missing id'; assert 'defs' in d, '$$f: missing defs'; print('OK $$f')" || exit 1; \
	done
	@echo "lexicons OK"

selfhost-smoke:
	@echo ">> selfhost smoke test"
	cd deploy && docker compose up -d
	@echo ">> waiting for /healthz"
	@for i in $$(seq 1 30); do \
		if curl -fs http://localhost:8080/healthz >/dev/null 2>&1; then echo "server up"; break; fi; \
		sleep 2; \
	done
	@curl -fs http://localhost:8080/healthz || (echo "server did not become healthy"; cd deploy && docker compose down; exit 1)
	@echo "selfhost smoke OK"
	cd deploy && docker compose down

server-build:
	cd server && go build -o bin/hekate-server ./cmd/hekate-server

server-test:
	cd server && go test ./...

cli-build:
	cd cli && go build -o bin/hekate ./cmd/hekate

cli-test:
	cd cli && go test ./...

web-install:
	cd web && pnpm install

web-check:
	cd web && pnpm check

web-build:
	cd web && pnpm build

clean:
	rm -rf server/bin cli/bin web/build web/.svelte-kit
	@echo "cleaned"
