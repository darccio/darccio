package buildsite

import (
	"bytes"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"dario.cat/site-tools/internal/vanity"
	"golang.org/x/net/html"
)

// finalizeNavigation changes only staged HTML navigation. Canonical head data,
// feeds, Markdown and agent contexts retain their production identifiers.
func finalizeNavigation(deploy string, documented func(string) bool) error {
	info, err := os.Lstat(deploy)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("navigation deploy must be a real directory")
	}
	rawVanity := map[string]bool{}
	for _, route := range vanity.Routes() {
		rawVanity[route.File()] = documented == nil || !documented(route.File())
	}
	var pages []markdownDiscoveryPage
	// Audit the entire inventory, including files that will not be rewritten,
	// before reading and planning changes. No symlink or special file is followed.
	if err := filepath.WalkDir(deploy, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("navigation refuses symlink %s", file)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("navigation refuses nonregular file %s", file)
		}
		relative, err := filepath.Rel(deploy, file)
		if err != nil {
			return err
		}
		if strings.EqualFold(filepath.Ext(relative), ".html") && !rawVanity[filepath.ToSlash(relative)] {
			pages = append(pages, markdownDiscoveryPage{file: file, mode: info.Mode()})
		}
		return nil
	}); err != nil {
		return err
	}
	var changes []markdownDiscoveryChange
	for _, page := range pages {
		relative, err := filepath.Rel(deploy, page.file)
		if err != nil {
			return err
		}
		body, err := readAgentContextFile(deploy, filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		normalized, err := navigationHTML(body)
		if err != nil {
			return fmt.Errorf("navigation %s: %w", relative, err)
		}
		if !bytes.Equal(body, normalized) {
			changes = append(changes, markdownDiscoveryChange{page: page, body: normalized})
		}
	}
	for _, change := range changes {
		if err := writeMarkdownDiscovery(change); err != nil {
			return err
		}
	}
	return nil
}

type navigationPatch struct {
	start, end int
	value      []byte
}
type navigationAttribute struct {
	start, end int
	quoted     bool
}
type navigationFrame struct {
	tag     string
	blocked bool
	hidden  bool
	foreign bool
}

// Tokenizer offsets preserve every byte outside real anchor attributes. The
// tokenizer treats quoted attributes, comments and script content as content,
// rather than searching their text for strings resembling href attributes.
func navigationHTML(body []byte) ([]byte, error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	var patches []navigationPatch
	var stack []navigationFrame
	offset, inBody := 0, false
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return nil, z.Err()
			}
			break
		}
		start := offset
		raw := bytes.Clone(z.Raw())
		offset += len(raw)
		token := z.Token()
		if kind == html.EndTagToken {
			if token.Data == "body" {
				inBody = false
			}
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == token.Data {
					stack = stack[:i]
					break
				}
			}
			continue
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		if token.Data == "body" {
			inBody = true
		}
		attrs := map[string]string{}
		for _, attr := range token.Attr {
			attrs[attr.Key] = attr.Val
		}
		blocked, hidden, foreign := false, false, false
		if len(stack) != 0 {
			parent := stack[len(stack)-1]
			blocked, hidden, foreign = parent.blocked, parent.hidden, parent.foreign
		}
		_, ownHidden := attrs["hidden"]
		style := strings.ToLower(strings.Join(strings.Fields(attrs["style"]), ""))
		hidden = hidden || ownHidden || strings.Contains(style, "display:none") || strings.Contains(style, "visibility:hidden") || strings.Contains(style, "visibility:collapse")
		_, inert := attrs["inert"]
		blocked = blocked || inert
		switch token.Data {
		case "head", "template":
			blocked = true
		case "svg", "math":
			foreign = true
		}
		if inBody && token.Data == "a" && !blocked {
			spans, err := navigationAttributes(raw)
			if err != nil {
				return nil, err
			}
			identityAnchor := navigationClass(attrs["class"], "u-uid") || navigationClass(attrs["class"], "u-url") || navigationClass(attrs["class"], "p-author") || navigationClass(attrs["class"], "h-card")
			href, hasHref := attrs["href"]
			if hasHref && !(hidden && identityAnchor) {
				target, internal, err := navigationTarget(href)
				if err != nil {
					return nil, err
				}
				uid := navigationClass(attrs["class"], "u-uid")
				rootUID := uid && strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//")
				if rootUID && !internal {
					target, err = navigationRootTarget(href)
					if err != nil {
						return nil, err
					}
				}
				if internal || rootUID {
					if internal {
						patches = append(patches, navigationValuePatch(start, spans["href"], target))
					}
					if uid {
						if foreign {
							return nil, errors.New("cannot move a canonical UID into a foreign-namespace anchor")
						}
						if kind == html.SelfClosingTagToken {
							return nil, errors.New("cannot move identity into a self-closing anchor")
						}
						var classes []string
						for _, class := range strings.Fields(attrs["class"]) {
							if class != "u-uid" {
								classes = append(classes, class)
							}
						}
						patches = append(patches, navigationValuePatch(start, spans["class"], strings.Join(classes, " ")))
						identityURL := href
						if rootUID {
							identityURL = "https://dario.cat" + target
						} else if strings.HasPrefix(strings.Trim(href, " \t\r\n\f"), "//") {
							identityURL = "https:" + strings.Trim(href, " \t\r\n\f")
						}
						identity := `<data class="u-uid" value="` + stdhtml.EscapeString(identityURL) + `" hidden></data>`
						patches = append(patches, navigationPatch{start + len(raw), start + len(raw), []byte(identity)})
					}
				}
			}
		}
		if kind == html.StartTagToken && !navigationVoid(token.Data) {
			stack = append(stack, navigationFrame{token.Data, blocked, hidden, foreign})
		}
	}
	if len(patches) == 0 {
		return body, nil
	}
	sort.SliceStable(patches, func(i, j int) bool { return patches[i].start < patches[j].start })
	var result bytes.Buffer
	previous := 0
	for _, patch := range patches {
		if patch.start < previous || patch.end < patch.start || patch.end > len(body) {
			return nil, errors.New("overlapping navigation byte patches")
		}
		result.Write(body[previous:patch.start])
		result.Write(patch.value)
		previous = patch.end
	}
	result.Write(body[previous:])
	return result.Bytes(), nil
}

func navigationClass(classes, wanted string) bool {
	for _, class := range strings.Fields(classes) {
		if class == wanted {
			return true
		}
	}
	return false
}

func navigationVoid(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

// Default ports belong to the site's HTTP/HTTPS origins. Protocol-relative
// links inherit the site's HTTPS origin; subdomains and other ports are external.
func navigationTarget(href string) (target string, internal bool, err error) {
	value := strings.Trim(href, " \t\r\n\f")
	// Browsers treat backslashes as separators in HTTP origins. Reject a
	// malformed self-host URL rather than letting it bypass the parsed-host check.
	if strings.Contains(value, "\\") && navigationSiteAuthority(strings.ReplaceAll(value, "\\", "/")) {
		return "", true, fmt.Errorf("unsafe internal navigation URL %q contains a backslash", href)
	}
	u, err := url.Parse(value)
	if err != nil {
		if navigationSiteAuthority(value) {
			return "", true, fmt.Errorf("unsafe internal navigation URL %q: %w", href, err)
		}
		return "", false, nil
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "" && strings.HasPrefix(value, "//") {
		scheme = "https"
	}
	if scheme != "http" && scheme != "https" || !strings.EqualFold(u.Hostname(), "dario.cat") {
		return "", false, nil
	}
	if u.User != nil || strings.Contains(value, "\\") {
		return "", true, fmt.Errorf("unsafe internal navigation URL %q has userinfo or a backslash", href)
	}
	port := "443"
	if scheme == "http" {
		port = "80"
	}
	if u.Port() != "" && u.Port() != port {
		return "", false, nil
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if err := navigationSafePath(u.Path, path); err != nil {
		return "", true, fmt.Errorf("unsafe internal navigation URL %q: %w", href, err)
	}
	return navigationURLSuffix(u, path, strings.Contains(value, "#")), true, nil
}

// Used only to diagnose malformed URLs that url.Parse cannot represent.
// Successful origin matching always uses the parsed Hostname and Port above.
func navigationSiteAuthority(value string) bool {
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "https://"):
		value = value[len("https://"):]
	case strings.HasPrefix(lower, "http://"):
		value = value[len("http://"):]
	case strings.HasPrefix(value, "//"):
		value = value[2:]
	default:
		return false
	}
	if i := strings.IndexAny(value, "/?#"); i >= 0 {
		value = value[:i]
	}
	if i := strings.LastIndex(value, "@"); i >= 0 {
		value = value[i+1:]
	}
	if i := strings.IndexByte(value, ':'); i >= 0 {
		value = value[:i]
	}
	return strings.EqualFold(value, "dario.cat")
}

func navigationRootTarget(href string) (string, error) {
	u, err := url.Parse(href)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "", fmt.Errorf("unsafe root identity URL %q", href)
	}
	if err := navigationSafePath(u.Path, u.EscapedPath()); err != nil {
		return "", fmt.Errorf("unsafe root identity URL %q: %w", href, err)
	}
	return navigationURLSuffix(u, u.EscapedPath(), strings.Contains(href, "#")), nil
}

func navigationSafePath(decoded, escaped string) error {
	if !strings.HasPrefix(escaped, "/") || strings.HasPrefix(escaped, "//") || strings.HasPrefix(decoded, "//") || strings.Contains(decoded, "\\") || strings.IndexFunc(decoded, unicode.IsControl) >= 0 {
		return errors.New("path must stay on the current origin")
	}
	return nil
}

func navigationURLSuffix(u *url.URL, path string, fragment bool) string {
	if u.ForceQuery || u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	if fragment {
		path += "#" + u.EscapedFragment()
	}
	return path
}

func navigationValuePatch(offset int, span navigationAttribute, value string) navigationPatch {
	escaped := stdhtml.EscapeString(value)
	if !span.quoted {
		escaped = `"` + escaped + `"`
	}
	return navigationPatch{offset + span.start, offset + span.end, []byte(escaped)}
}

// Scan only a tokenizer-delimited tag. Quoted values are consumed as one field,
// so a title containing 'href=...' cannot be mistaken for the real attribute.
func navigationAttributes(raw []byte) (map[string]navigationAttribute, error) {
	space := func(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' }
	spans := map[string]navigationAttribute{}
	seenIdentity := map[string]bool{}
	i := 1
	for i < len(raw) && !space(raw[i]) && raw[i] != '>' && raw[i] != '/' {
		i++
	}
	for i < len(raw) {
		for i < len(raw) && (space(raw[i]) || raw[i] == '/') {
			i++
		}
		if i == len(raw) || raw[i] == '>' {
			break
		}
		start := i
		for i < len(raw) && !space(raw[i]) && raw[i] != '=' && raw[i] != '>' && raw[i] != '/' {
			i++
		}
		if i == start {
			return nil, errors.New("unsupported anchor attribute syntax")
		}
		name := strings.ToLower(string(raw[start:i]))
		if name == "href" || name == "class" {
			if seenIdentity[name] {
				return nil, fmt.Errorf("ambiguous anchor with duplicate %s attributes", name)
			}
			seenIdentity[name] = true
		}
		for i < len(raw) && space(raw[i]) {
			i++
		}
		if i == len(raw) || raw[i] != '=' {
			continue
		}
		i++
		for i < len(raw) && space(raw[i]) {
			i++
		}
		span := navigationAttribute{start: i}
		if i < len(raw) && (raw[i] == '"' || raw[i] == '\'') {
			quote := raw[i]
			i++
			span.start, span.quoted = i, true
			for i < len(raw) && raw[i] != quote {
				i++
			}
			span.end = i
			if i == len(raw) {
				return nil, errors.New("unterminated anchor attribute")
			}
			i++
		} else {
			for i < len(raw) && !space(raw[i]) && raw[i] != '>' {
				i++
			}
			span.end = i
		}
		spans[name] = span
	}
	return spans, nil
}
