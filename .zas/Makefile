serve:
	hivemind
publish:
	GOBIN=/tmp go install github.com/imdario/zas@latest
	/tmp/zas generate
	echo '<meta name="go-import" content="dario.cat/mergo git https://github.com/imdario/mergo">' > .zas/deploy/mergo/index.html