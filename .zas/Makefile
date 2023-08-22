serve:
	hivemind

publish:
	GOBIN=/tmp go install github.com/darccio/zas@latest
	/tmp/zas generate
	cat mergo/index.html > .zas/deploy/mergo/index.html
	cat xdr/index.html > .zas/deploy/xdr/index.html
