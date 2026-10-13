package check

import (
	"strings"
	"testing"
)

const englishNavigationFixture = `<html lang="en"><body>
<a class="skip-link" lang="en" href="#main-content">Skip to content</a>
<header><a class="site-logo" href="/en/">Dario Castañé</a>
<nav class="primary-navigation" lang="en" aria-label="Main navigation">
<a href="/en/">Home</a><a href="/en/posts/" aria-current="page">Articles</a>
<a href="/en/projects/">Projects</a><a href="/en/talks/">Talks</a><a href="/en/contact/">Contact</a>
</nav><nav class="site-language-nav" aria-label="Site language">
<a href="/" lang="ca" hreflang="ca">Català</a><a href="/en/" lang="en" hreflang="en" aria-current="true">English</a></nav></header>
<main id="main-content">Articles</main>
<footer><p><a href="/en/feeds/">Feeds</a><a href="/en/slashes/">All pages</a><a href="/en/colophon/">About this site</a></p></footer>
</body></html>`

func TestEnglishNavigationRetainsTheLocale(t *testing.T) {
	for _, path := range []string{"/en/posts/", "/en/posts/example/", "/en/feeds/"} {
		doc, _ := parsed(t, englishNavigationFixture, homeURL+strings.TrimPrefix(path, "/"))
		if err := validateLocaleNavigation(doc, path); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"/en/", "/en/now/"} {
		input := strings.Replace(englishNavigationFixture, `href="/en/posts/" aria-current="page"`, `href="/en/posts/"`, 1)
		if path == "/en/" {
			input = strings.Replace(input, `href="/en/">Home`, `href="/en/" aria-current="page">Home`, 1)
		}
		doc, _ := parsed(t, input, homeURL+strings.TrimPrefix(path, "/"))
		if err := validateLocaleNavigation(doc, path); err != nil {
			t.Fatal(err)
		}
	}
	for _, mutation := range [][2]string{
		{`<html lang="en">`, `<html lang="ca">`},
		{`href="/en/posts/"`, `href="/posts/"`},
		{`href="/en/projects/"`, `href="https://dario.cat/en/projects/"`},
		{`class="site-logo" href="/en/"`, `class="site-logo" href="/"`},
		{`Skip to content`, `Salta al contingut`},
		{`aria-label="Main navigation"`, `aria-label="Navegació principal"`},
		{`href="/en/posts/" aria-current="page"`, `href="/en/posts/"`},
		{`href="/en/talks/"`, `href="/en/talks/" aria-current="page"`},
		{`href="/en/feeds/"`, `href="/feeds/"`},
		{`hreflang="en"`, `hreflang="ca"`},
	} {
		t.Run(mutation[0], func(t *testing.T) {
			doc, _ := parsed(t, strings.Replace(englishNavigationFixture, mutation[0], mutation[1], 1), homeURL+"en/posts/")
			if err := validateLocaleNavigation(doc, "/en/posts/"); err == nil {
				t.Fatal("locale navigation regression was accepted")
			}
		})
	}
}
