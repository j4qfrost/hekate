# hekate — top-level Makefile
# Wraps per-subproject toolchains so contributors don't have to memorise them.

.PHONY: help dev test lex-validate selfhost-smoke server-build server-test cli-build cli-test web-install web-check web-build clean companion-up companion-down companion-load companion-verify obs-up obs-down obs-status

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
	@echo "  make companion-up     bring up the RisingWave companion stack (deploy/companion/)"
	@echo "  make companion-down   tear down the companion stack"
	@echo "  make companion-load   apply RisingWave SQL + run hekate-fixturegen (--scenario=skewed)"
	@echo "  make companion-verify run W3 collision-rule correctness check vs. Postgres ground truth"
	@echo ""
	@echo "  make obs-up           bring up the LGTM observability stack (deploy/observability/)"
	@echo "  make obs-down         tear down the observability stack"
	@echo "  make obs-status       show observability stack health + URLs"
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
	rm -rf server/bin cli/bin web/build web/.svelte-kit companion/bin
	@echo "cleaned"

# Companion stream-processing stack (RisingWave). Self-contained; does NOT
# join `selfhost-smoke`. See docs/adr/0004-companion-stream-engine.md.

COMPANION_COMPOSE := docker compose -f deploy/companion/docker-compose.yml
PG_DSN_COMPANION  := postgres://hekate:hekate@localhost:5433/hekate?sslmode=disable
RW_DSN_COMPANION  := postgres://root@localhost:4566/dev?sslmode=disable

companion-up:
	@echo ">> bringing up companion stack (postgres + goose + risingwave)"
	$(COMPANION_COMPOSE) up -d
	@echo ">> waiting for risingwave to accept connections on :4566"
	@for i in $$(seq 1 60); do \
		if $(COMPANION_COMPOSE) exec -T risingwave sh -c 'echo > /dev/tcp/127.0.0.1/4566' >/dev/null 2>&1; then \
			echo "risingwave up"; exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "risingwave did not become reachable"; exit 1

companion-down:
	$(COMPANION_COMPOSE) down -v

companion-load: companion-up
	@echo ">> applying companion SQL to risingwave"
	$(COMPANION_COMPOSE) run --rm \
	  -v "$(CURDIR)/companion/sql:/sql:ro" \
	  --entrypoint sh postgres -c \
	  'set -e; for f in /sql/*.sql; do echo "applying $$f"; psql "postgres://root@risingwave:4566/dev?sslmode=disable" -v ON_ERROR_STOP=1 -f "$$f"; done'
	@echo ">> running hekate-fixturegen (skewed)"
	cd companion && go run ./fixturegen/cmd/hekate-fixturegen \
	  --dsn "$(PG_DSN_COMPANION)" \
	  --scenario=skewed \
	  --num-venues=3 --slots-per-venue=4 --collision-rate=0.6

companion-verify:
	cd companion && go run ./verify/cmd/hekate-verify-w3 \
	  --postgres-dsn "$(PG_DSN_COMPANION)" \
	  --risingwave-dsn "$(RW_DSN_COMPANION)"

# LGTM observability stack (Loki + Grafana + Tempo + Mimir via grafana/otel-lgtm).
# Self-contained; does NOT join `selfhost-smoke`. See docs/adr/0005-lgtm-observability-stack.md
# and docs/OBSERVABILITY.md.

OBS_COMPOSE := docker compose -f deploy/observability/docker-compose.yml

obs-up:
	@echo ">> bringing up LGTM stack (grafana/otel-lgtm)"
	$(OBS_COMPOSE) up -d
	@echo ">> waiting for Grafana on :3001"
	@for i in $$(seq 1 30); do \
		if curl -fs http://localhost:3001/api/health >/dev/null 2>&1; then \
			echo "Grafana up: http://localhost:3001/d/hekate-overview"; \
			echo "OTLP endpoint: localhost:4317 (gRPC) / localhost:4318 (HTTP)"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Grafana did not become healthy in 60s; check 'docker logs hekate-lgtm'"; exit 1

obs-down:
	$(OBS_COMPOSE) down -v

obs-status:
	@$(OBS_COMPOSE) ps
	@echo ""
	@echo "Grafana:  http://localhost:3001/d/hekate-overview  (anonymous Admin)"
	@echo "OTLP:     localhost:4317 (gRPC) / localhost:4318 (HTTP)"
	@echo "Loki:     http://localhost:3100"
	@echo "Tempo:    http://localhost:3200"
	@echo "Mimir:    http://localhost:9009"
