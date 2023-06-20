serve:
	hivemind

publish:
	GOBIN=/tmp go install github.com/imdario/zas@latest
	/tmp/zas generate
	cat mergo/index.html > .zas/deploy/mergo/index.html
	cat xdr/index.html > .zas/deploy/xdr/index.html
