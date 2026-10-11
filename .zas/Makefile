ZAS ?= zas
ZAS_VERSION := v0.0.7
ZAS_BIN_DIR ?= /tmp/dario-site-zas

.PHONY: serve generate publish check-vanity check-vanity-live test

serve:
	hivemind .zas/Procfile

generate:
	go -C .tools/vanity run ./cmd/build-site -root ../.. -zas "$(ZAS)"

publish:
	mkdir -p "$(ZAS_BIN_DIR)"
	GOBIN="$(abspath $(ZAS_BIN_DIR))" go install github.com/darccio/zas/cmd/zas@$(ZAS_VERSION)
	go -C .tools/vanity run ./cmd/build-site -root ../.. -zas "$(abspath $(ZAS_BIN_DIR))/zas"

check-vanity:
	go -C .tools/vanity run ./cmd/vanity -dir ../../.zas/deploy

check-vanity-live:
	go -C .tools/vanity run ./cmd/vanity -live -base-url https://dario.cat -timeout 10s

test:
	go -C .tools/vanity test ./...
