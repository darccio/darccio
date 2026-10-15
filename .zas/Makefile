ZAS ?= zas
ZAS_VERSION := v0.0.10
ZAS_BIN_DIR ?= /tmp/dario-site-zas

.PHONY: serve generate publish check-vanity check-vanity-live check-microformats check-consistency test

serve:
	hivemind .zas/Procfile

generate:
	go -C .tools/vanity run ./cmd/build-site -root ../.. -zas "$(ZAS)" -check-microformats -check-consistency

publish:
	@set -e; \
	site_is_shallow=$$(git rev-parse --is-shallow-repository); \
	if [ "$$site_is_shallow" = true ]; then \
		git fetch --unshallow --no-tags origin; \
		test "$$(git rev-parse --is-shallow-repository)" = false; \
	fi
	mkdir -p "$(ZAS_BIN_DIR)"
	GOBIN="$(abspath $(ZAS_BIN_DIR))" go install github.com/darccio/zas/cmd/zas@$(ZAS_VERSION)
	go -C .tools/vanity run ./cmd/build-site -root ../.. -zas "$(abspath $(ZAS_BIN_DIR))/zas" -check-microformats -check-consistency

check-vanity:
	go -C .tools/vanity run ./cmd/vanity -dir ../../.zas/deploy

check-vanity-live:
	go -C .tools/vanity run ./cmd/vanity -live -base-url https://dario.cat -timeout 10s

check-microformats:
	go -C .tools/microformats run -mod=readonly ./cmd/microformats -root ../.. -dir ../../.zas/deploy

check-consistency:
	@plan=$$(mktemp /tmp/dario-standard-plan.XXXXXX); \
	trap 'rm -f "$$plan"' EXIT; \
	go -C .tools/atproto run ./cmd/documents -root ../.. -plan "$$plan" > /dev/null && \
	go -C .tools/microformats run -mod=readonly ./cmd/consistency -root ../.. -dir ../../.zas/deploy -standard-plan "$$plan"

test:
	go -C .tools/vanity test ./...
	go -C .tools/microformats test -mod=readonly ./...
	go -C .tools/atproto test ./...
	node --test .tools/http/worker.test.mjs
	bash .tools/docs/open-update-pr_test.sh
	python3 .tools/atproto/open-publication-pr_test.py
	python3 .tools/atproto/deployed-revision_test.py
	python3 .tools/atproto/open-documents-pr_test.py
