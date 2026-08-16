serve:
	hivemind .zas/Procfile

generate:
	zas generate -full

publish:
	GOBIN=/tmp go install github.com/darccio/zas/cmd/zas@v0.0.6
	/tmp/zas generate
	cat mergo/index.html > .zas/deploy/mergo/index.html
	cat darna/index.html > .zas/deploy/darna/index.html
	cat errors/index.html > .zas/deploy/errors/index.html
	cat xdr/index.html > .zas/deploy/xdr/index.html
	cat awaken/index.html > .zas/deploy/awaken/index.html
	cat x/index.html > .zas/deploy/x/index.html
