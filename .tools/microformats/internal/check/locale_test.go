package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const englishNavigationFixture = `<html lang="en"><body>
<a class="skip-link" lang="en" href="#main-content">Skip to content</a>
<header><a class="site-logo" href="/en/">Dario Castañé</a>
<nav class="primary-navigation" lang="en" aria-label="Main navigation">
<a href="/en/">Home</a><a href="/en/posts/">Articles</a>
<a href="/en/projects/">Projects</a><a href="/en/talks/">Talks</a><a href="/en/contact/">Contact</a>
</nav><nav class="site-language-nav" aria-label="Site language">LANGUAGE LINKS</nav></header>
<main id="main-content">Articles</main>
<footer><p><a href="/en/feeds/">Feeds</a><a href="/en/slashes/">All pages</a><a href="/en/colophon/">About this site</a></p></footer>
</body></html>`

func languageLinksFixture(current string, targets map[string]string) string {
	var links strings.Builder
	for _, locale := range []struct{ code, label string }{{"ca", "Català"}, {"en", "English"}, {"es", "Castellano"}} {
		if target := targets[locale.code]; target != "" {
			fmt.Fprintf(&links, `<a href="%s" lang="%s" hreflang="%s"`, target, locale.code, locale.code)
			if locale.code == current {
				links.WriteString(` aria-current="page"`)
			}
			fmt.Fprintf(&links, ` aria-label="%s">%s</a>`, locale.label, strings.ToUpper(locale.code))
		}
	}
	return links.String()
}

func englishPageFixture(section string, targets map[string]string) string {
	input := strings.Replace(englishNavigationFixture, "LANGUAGE LINKS", languageLinksFixture("en", targets), 1)
	if section != "" {
		labels := map[string]string{"/en/": "Home", "/en/posts/": "Articles", "/en/projects/": "Projects", "/en/talks/": "Talks", "/en/contact/": "Contact"}
		before := fmt.Sprintf(`href="%s">%s`, section, labels[section])
		input = strings.Replace(input, before, fmt.Sprintf(`href="%s" aria-current="page">%s`, section, labels[section]), 1)
	}
	return input
}

func TestEnglishNavigationRetainsTheLocale(t *testing.T) {
	for _, test := range []struct {
		path, section string
		targets       map[string]string
	}{
		{"/en/posts/", "/en/posts/", map[string]string{"ca": "/posts/", "en": "/en/posts/", "es": "/es/posts/"}},
		{"/en/posts/example/", "/en/posts/", map[string]string{"ca": "/", "en": "/en/posts/example/"}},
		{"/en/feeds/", "/en/posts/", map[string]string{"ca": "/feeds/", "en": "/en/feeds/"}},
		{"/en/", "/en/", map[string]string{"ca": "/", "en": "/en/"}},
		{"/en/now/", "", map[string]string{"ca": "/now/", "en": "/en/now/"}},
		{"/en/talks/", "/en/talks/", map[string]string{"ca": "/talks/", "en": "/en/talks/"}},
		{"/cv/", "", map[string]string{"ca": "/", "en": "/cv/"}},
		{"/zas/templates/", "/en/projects/", map[string]string{"ca": "/", "en": "/zas/templates/"}},
	} {
		t.Run(test.path, func(t *testing.T) {
			doc, _ := parsed(t, englishPageFixture(test.section, test.targets), homeURL+strings.TrimPrefix(test.path, "/"))
			if err := validateLocaleNavigation(doc, test.path); err != nil {
				t.Fatal(err)
			}
		})
	}
	input := englishPageFixture("/en/posts/", map[string]string{"ca": "/posts/", "en": "/en/posts/", "es": "/es/posts/"})
	for _, mutation := range []struct{ name, before, after string }{
		{"document language", `<html lang="en">`, `<html lang="ca">`},
		{"primary locale", `href="/en/posts/" aria-current="page"`, `href="/posts/" aria-current="page"`},
		{"primary absolute URL", `href="/en/projects/"`, `href="https://dario.cat/en/projects/"`},
		{"brand", `class="site-logo" href="/en/"`, `class="site-logo" href="/"`},
		{"skip language", `Skip to content`, `Salta al contingut`},
		{"primary label", `aria-label="Main navigation"`, `aria-label="Navegació principal"`},
		{"missing section", `href="/en/posts/" aria-current="page"`, `href="/en/posts/"`},
		{"multiple sections", `href="/en/talks/"`, `href="/en/talks/" aria-current="page"`},
		{"footer locale", `href="/en/feeds/"`, `href="/feeds/"`},
		{"language absolute URL", `href="/posts/" lang="ca"`, `href="https://dario.cat/posts/" lang="ca"`},
		{"language protocol-relative URL", `href="/posts/" lang="ca"`, `href="//dario.cat/posts/" lang="ca"`},
		{"language relative URL", `href="/posts/" lang="ca"`, `href="posts/" lang="ca"`},
		{"language query", `href="/posts/" lang="ca"`, `href="/posts/?lang=ca" lang="ca"`},
		{"inconsistent language", `hreflang="en"`, `hreflang="ca"`},
		{"duplicate language", `lang="es" hreflang="es"`, `lang="ca" hreflang="ca"`},
		{"incorrect visible code", `>EN</a>`, `>FR</a>`},
		{"full name replaces compact code", `>EN</a>`, `>English</a>`},
		{"missing accessible full name", ` aria-label="English"`, ``},
		{"wrong accessible full name", `aria-label="English"`, `aria-label="Anglès"`},
		{"code replaces accessible full name", `aria-label="English"`, `aria-label="EN"`},
		{"wrong Catalan accessible full name", `aria-label="Català"`, `aria-label="Catalan"`},
		{"wrong Spanish accessible full name", `aria-label="Castellano"`, `aria-label="Español"`},
		{"language nav label", `aria-label="Site language"`, `aria-label="Idioma del web"`},
		{"old current value", `lang="en" hreflang="en" aria-current="page"`, `lang="en" hreflang="en" aria-current="true"`},
		{"missing current language", `lang="en" hreflang="en" aria-current="page"`, `lang="en" hreflang="en"`},
		{"multiple current languages", `lang="ca" hreflang="ca"`, `lang="ca" hreflang="ca" aria-current="page"`},
		{"current language leaves page", `href="/en/posts/" lang="en"`, `href="/en/" lang="en"`},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			if !strings.Contains(input, mutation.before) {
				t.Fatal("mutation does not match the fixture")
			}
			doc, _ := parsed(t, strings.Replace(input, mutation.before, mutation.after, 1), homeURL+"en/posts/")
			if err := validateLocaleNavigation(doc, "/en/posts/"); err == nil {
				t.Fatal("locale navigation regression was accepted")
			}
		})
	}
}

func basicLanguagePageFixture(language string, targets map[string]string) string {
	labels := map[string]string{"ca": "Idioma del web", "es": "Idioma del sitio"}
	return fmt.Sprintf(`<html lang="%s"><body><header><a class="site-logo" href="/">Dario Castañé</a><nav class="site-language-nav" aria-label="%s">%s</nav></header></body></html>`, language, labels[language], languageLinksFixture(language, targets))
}

func localeDeploymentFixture(t *testing.T) string {
	t.Helper()
	deploy := t.TempDir()
	for _, slug := range []string{"", "about", "now", "projects", "talks", "contact", "hello", "feeds", "slashes", "colophon", "ai", "languages", "elsewhere", "sponsor", "uses", "blogroll", "changelog", "posts"} {
		pagePath := "/en/"
		if slug != "" {
			pagePath += slug + "/"
		}
		sections := map[string]string{"": "/en/", "about": "/en/", "projects": "/en/projects/", "talks": "/en/talks/", "contact": "/en/contact/", "hello": "/en/contact/", "feeds": "/en/posts/", "posts": "/en/posts/"}
		put(t, deploy, strings.TrimPrefix(pagePath, "/")+"index.html", englishPageFixture(sections[slug], map[string]string{"ca": "/", "en": pagePath}))
	}
	put(t, deploy, "index.html", basicLanguagePageFixture("ca", map[string]string{"ca": "/", "en": "/en/"}))
	put(t, deploy, "posts/ca/index.html", basicLanguagePageFixture("ca", map[string]string{"ca": "/posts/ca/", "en": "/en/"}))
	put(t, deploy, "cv/index.html", englishPageFixture("", map[string]string{"ca": "/", "en": "/cv/"}))
	put(t, deploy, "es/posts/standalone/index.html", basicLanguagePageFixture("es", map[string]string{"ca": "/", "en": "/en/", "es": "/es/posts/standalone/"}))
	put(t, deploy, "cocoon/index.html", `<meta name="go-import" content="dario.cat/cocoon git https://github.com/darccio/cocoon">`)
	targets := map[string]string{"ca": "/posts/example/", "en": "/en/posts/translated-example/", "es": "/es/posts/ejemplo/"}
	var sitemap strings.Builder
	sitemap.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">`)
	for _, language := range []string{"ca", "en", "es"} {
		pagePath := targets[language]
		input := basicLanguagePageFixture(language, targets)
		if language == "en" {
			input = englishPageFixture("/en/posts/", targets)
		}
		put(t, deploy, strings.TrimPrefix(pagePath, "/")+"index.html", input)
		fmt.Fprintf(&sitemap, `<url><loc>%s%s</loc>`, homeURL, strings.TrimPrefix(pagePath, "/"))
		for _, code := range []string{"ca", "en", "es"} {
			fmt.Fprintf(&sitemap, `<xhtml:link rel="alternate" hreflang="%s" href="%s%s"></xhtml:link>`, code, homeURL, strings.TrimPrefix(targets[code], "/"))
		}
		sitemap.WriteString(`</url>`)
	}
	sitemap.WriteString(`</urlset>`)
	put(t, deploy, "sitemap.xml", sitemap.String())
	return deploy
}

func TestLocaleNavigationChecksDeployedCounterparts(t *testing.T) {
	if err := validateEnglishNavigation(localeDeploymentFixture(t)); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, file, before, after, want string }{
		{"dead destination", "en/about/index.html", `href="/" lang="ca"`, `href="/missing/" lang="ca"`, "no deployed page"},
		{"destination language", "en/about/index.html", `href="/" lang="ca"`, `href="/en/now/" lang="ca"`, "uses language"},
		{"translation replaced by home", "en/posts/translated-example/index.html", `href="/posts/example/" lang="ca"`, `href="/" lang="ca"`, "must point to /posts/example/"},
		{"omitted Spanish translation", "en/posts/translated-example/index.html", `<a href="/es/posts/ejemplo/" lang="es" hreflang="es" aria-label="Castellano">ES</a>`, "", "omitted or invented"},
		{"invented Spanish translation", "en/about/index.html", `</nav></header>`, `<a href="/es/posts/standalone/" lang="es" hreflang="es" aria-label="Castellano">ES</a></nav></header>`, "omitted or invented"},
		{"English language lost", "en/about/index.html", `<html lang="en">`, `<html lang="ca">`, "lost its English language"},
		{"Spanish self lost", "es/posts/standalone/index.html", `href="/es/posts/standalone/"`, `href="/es/posts/"`, "current es language link"},
	} {
		t.Run(test.name, func(t *testing.T) {
			deploy := localeDeploymentFixture(t)
			file := filepath.Join(deploy, filepath.FromSlash(test.file))
			input, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(input), test.before) {
				t.Fatal("mutation does not match the fixture")
			}
			put(t, deploy, test.file, strings.Replace(string(input), test.before, test.after, 1))
			if err := validateEnglishNavigation(deploy); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("wanted %q rejection, got %v", test.want, err)
			}
		})
	}
	t.Run("missing published counterpart", func(t *testing.T) {
		deploy := localeDeploymentFixture(t)
		if err := os.Remove(filepath.Join(deploy, "es/posts/ejemplo/index.html")); err != nil {
			t.Fatal(err)
		}
		if err := validateEnglishNavigation(deploy); err == nil || !strings.Contains(err.Error(), "no deployed page") {
			t.Fatalf("missing published counterpart accepted: %v", err)
		}
	})
	t.Run("missing required English page", func(t *testing.T) {
		deploy := localeDeploymentFixture(t)
		if err := os.Remove(filepath.Join(deploy, "en/about/index.html")); err != nil {
			t.Fatal(err)
		}
		if err := validateEnglishNavigation(deploy); err == nil || !strings.Contains(err.Error(), "missing translated English page") {
			t.Fatalf("missing English section accepted: %v", err)
		}
	})
	t.Run("sitemap current counterpart disagrees", func(t *testing.T) {
		deploy := localeDeploymentFixture(t)
		input, err := os.ReadFile(filepath.Join(deploy, "sitemap.xml"))
		if err != nil {
			t.Fatal(err)
		}
		marker := `<url><loc>https://dario.cat/en/posts/translated-example/</loc>`
		before, entry, found := strings.Cut(string(input), marker)
		if !found {
			t.Fatal("English sitemap fixture entry missing")
		}
		entry = strings.Replace(entry, `hreflang="en" href="https://dario.cat/en/posts/translated-example/"`, `hreflang="en" href="https://dario.cat/en/about/"`, 1)
		put(t, deploy, "sitemap.xml", before+marker+entry)
		if err := validateEnglishNavigation(deploy); err == nil || !strings.Contains(err.Error(), "sitemap current-language counterpart") {
			t.Fatalf("inconsistent sitemap self counterpart accepted: %v", err)
		}
	})
}
