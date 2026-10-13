package buildsite

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// Check the assembled files rather than HTTP status codes: Pages' homepage
// fallback can return 200 text/html for a missing image. Remote images remain
// outside this offline check, so the build never depends on their hosts.
func checkLocalImages(deploy string) error {
	return filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(file), ".html") {
			return nil
		}
		input, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		doc, err := html.Parse(bytes.NewReader(input))
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(deploy, file)
		if err != nil {
			return err
		}
		page := &url.URL{Scheme: "https", Host: "dario.cat", Path: "/" + filepath.ToSlash(relative)}
		var visit func(*html.Node) error
		visit = func(node *html.Node) error {
			if node.Type == html.ElementNode {
				attrs := make(map[string]string)
				for _, attr := range node.Attr {
					if _, exists := attrs[attr.Key]; !exists {
						attrs[attr.Key] = attr.Val
					}
				}
				var src string
				switch node.Data {
				case "img":
					src = attrs["src"]
				case "meta":
					if attrs["property"] == "og:image" {
						src = attrs["content"]
					}
				}
				if src != "" {
					if err := checkLocalImage(deploy, page, src); err != nil {
						return fmt.Errorf("%s: %w", relative, err)
					}
				}
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if err := visit(child); err != nil {
					return err
				}
			}
			return nil
		}
		return visit(doc)
	})
}

func checkLocalImage(deploy string, page *url.URL, src string) error {
	ref, err := url.Parse(src)
	if err != nil {
		return fmt.Errorf("invalid image URL %q: %w", src, err)
	}
	target := page.ResolveReference(ref)
	port := target.Port()
	defaultPort := port == "" || (target.Scheme == "https" && port == "443") || (target.Scheme == "http" && port == "80")
	if (target.Scheme != "https" && target.Scheme != "http") || !strings.EqualFold(target.Hostname(), "dario.cat") || !defaultPort {
		return nil
	}
	file := filepath.Join(deploy, filepath.FromSlash(strings.TrimPrefix(target.Path, "/")))
	relative, err := filepath.Rel(deploy, file)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("local image %q escapes the deployment directory", src)
	}
	info, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("local image %q has no deployed file: %w", src, err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("local image %q is not a nonempty regular file", src)
	}
	return nil
}
