serve:
	hivemind .zas/Procfile

generate:
	zas generate -full

publish:
	GOBIN=/tmp go install github.com/darccio/zas@v0.0.4
	/tmp/zas generate
	cat mergo/index.html > .zas/deploy/mergo/index.html
	cat errors/index.html > .zas/deploy/errors/index.html
	cat xdr/index.html > .zas/deploy/xdr/index.html
