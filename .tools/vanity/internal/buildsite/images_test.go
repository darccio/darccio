package buildsite

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalImageURLsResolveAgainstTheDeployedPage(t *testing.T) {
	deploy := t.TempDir()
	put(t, deploy, "static/images/cover & sky.jpg", "image fixture")
	put(t, deploy, "posts/example/photo.png", "image fixture")
	put(t, deploy, "posts/example/index.html", `<html><head>
<meta property="og:image" content="https://dario.cat/static/images/cover%20%26%20sky.jpg">
</head><body>
<img src="/static/images/cover%20%26%20sky.jpg?size=large">
<img src="photo.png"><img src="../../static/images/cover%20%26%20sky.jpg">
<img src="//dario.cat/static/images/cover%20%26%20sky.jpg">
<img src="http://dario.cat/static/images/cover%20%26%20sky.jpg">
<img src="https://dario.cat:443/static/images/cover%20%26%20sky.jpg">
<img src="http://dario.cat:80/static/images/cover%20%26%20sky.jpg">
<img src="https://remote.example/missing.png"><img src="data:image/png;base64,AA==">
<!-- <img src="/comments-are-not-images.png"> -->
<script type="application/ld+json">{"text":"<img src='/json-is-not-an-image.png'>"}</script>
</body></html>`)
	if err := checkLocalImages(deploy); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(deploy, "posts/example/photo.png")); err != nil {
		t.Fatal(err)
	}
	if err := checkLocalImages(deploy); err == nil || !strings.Contains(err.Error(), "photo.png") {
		t.Fatalf("missing relative image was accepted: %v", err)
	}
}

func TestLocalImagesRejectMissingFilesAndHomepageFallbacks(t *testing.T) {
	for _, source := range []string{
		`<img src="/images/missing.png">`,
		`<meta property="og:image" content="https://dario.cat/images/missing.png">`,
		`<img src="https://dario.cat/images/missing.png">`,
		`<img src="http://dario.cat/images/missing.png">`,
		`<img src="https://dario.cat:443/images/missing.png">`,
		`<img src="/images/missing.png" src="/images/valid.png">`,
		`<img src="/%2e%2e/outside.png">`,
		`<img src="/images/empty.png">`,
		`<img src="/images/">`,
	} {
		t.Run(source, func(t *testing.T) {
			deploy := t.TempDir()
			put(t, deploy, "index.html", "homepage fallback")
			put(t, deploy, "posts/example/index.html", source)
			put(t, deploy, "images/empty.png", "")
			put(t, deploy, "images/valid.png", "image fixture")
			if err := checkLocalImages(deploy); err == nil {
				t.Fatal("missing or unusable image asset was accepted")
			}
		})
	}
}

func TestMissingImagePreventsDeploymentPromotion(t *testing.T) {
	root := sourceTree(t)
	script := generator(t, ordinaryOutput+`
printf '<html><body><img src="/images/missing.png"></body></html>' > .zas/deploy/index.html
`)
	err := Build(context.Background(), Options{Root: root, Zas: script, Output: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "assembled image assets rejected") {
		t.Fatalf("missing image did not reject the build: %v", err)
	}
	previous, err := os.ReadFile(filepath.Join(root, ".zas/deploy/previous.txt"))
	if err != nil || string(previous) != "last successful deployment" {
		t.Fatalf("missing image replaced previous deployment: %q %v", previous, err)
	}
}
