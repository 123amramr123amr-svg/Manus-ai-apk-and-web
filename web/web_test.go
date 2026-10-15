package web

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// These tests keep the hand-written frontend consistent: every selector used
// by app.js must exist in index.html, every translation key must be present in
// both languages, and every locally referenced asset must be embedded.

var (
	idInHTML         = regexp.MustCompile(`id="([A-Za-z0-9_-]+)"`)
	selectorInJS     = regexp.MustCompile(`\$\('#([A-Za-z0-9_-]+)'\)`)
	keyInHTML        = regexp.MustCompile(`data-i18n(?:-placeholder|-title)?="([A-Za-z0-9_.-]+)"`)
	suggestionInHTML = regexp.MustCompile(`data-suggestion="([A-Za-z0-9_.-]+)"`)
	keyInJS          = regexp.MustCompile(`(?:^|[^A-Za-z0-9_$.])t\('([A-Za-z0-9_.-]+)'(?:,|\))`)
	keyInDict        = regexp.MustCompile(`"([A-Za-z0-9_.-]+)":`)
	assetInHTML      = regexp.MustCompile(`(?:src|href)="(/[^"]+)"`)
)

func readAsset(t *testing.T, name string) string {
	t.Helper()
	return readFrom(t, StaticFS(), name)
}

func readFrom(t *testing.T, fsys fs.FS, name string) string {
	t.Helper()
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func TestSelectorsExistInMarkup(t *testing.T) {
	html := readAsset(t, "index.html")
	script := readAsset(t, "app.js")

	ids := map[string]bool{}
	for _, match := range idInHTML.FindAllStringSubmatch(html, -1) {
		ids[match[1]] = true
	}

	missing := map[string]bool{}
	for _, match := range selectorInJS.FindAllStringSubmatch(script, -1) {
		if !ids[match[1]] {
			missing[match[1]] = true
		}
	}
	if len(missing) > 0 {
		t.Fatalf("app.js references elements missing from index.html: %v", sortedKeys(missing))
	}
}

func TestTranslationKeysAreComplete(t *testing.T) {
	html := readAsset(t, "index.html")
	script := readAsset(t, "app.js")
	dictionary := readAsset(t, "i18n.js")

	languages := map[string]map[string]bool{}
	for _, language := range []string{"ar", "en"} {
		block := languageBlock(t, dictionary, language)
		keys := map[string]bool{}
		for _, match := range keyInDict.FindAllStringSubmatch(block, -1) {
			keys[match[1]] = true
		}
		if len(keys) == 0 {
			t.Fatalf("no keys found for language %q", language)
		}
		languages[language] = keys
	}

	used := map[string]bool{}
	for _, match := range keyInHTML.FindAllStringSubmatch(html, -1) {
		used[match[1]] = true
	}
	for _, match := range keyInJS.FindAllStringSubmatch(script, -1) {
		if !strings.HasSuffix(match[1], ".") { // dynamic prefixes like t('activity.' + kind)
			used[match[1]] = true
		}
	}

	for language, keys := range languages {
		for key := range used {
			if !keys[key] {
				t.Errorf("translation %q is missing from the %s dictionary", key, language)
			}
		}
	}

	// Every key of the reference language must exist in the other one too.
	for key := range languages["ar"] {
		if !languages["en"][key] {
			t.Errorf("key %q exists in ar but not in en", key)
		}
	}
}

func TestReferencedAssetsAreEmbedded(t *testing.T) {
	html := readAsset(t, "index.html")
	assets := StaticFS()

	seen := map[string]bool{}
	for _, match := range assetInHTML.FindAllStringSubmatch(html, -1) {
		path := strings.TrimPrefix(match[1], "/")
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		if _, err := fs.Stat(assets, path); err != nil {
			t.Errorf("index.html references %q which is not embedded: %v", path, err)
		}
	}
	if len(seen) < 4 {
		t.Fatalf("expected several embedded assets, found %d", len(seen))
	}
}

func TestServiceWorkerAndManifestArePresent(t *testing.T) {
	assets := StaticFS()
	for _, name := range []string{
		"index.html", "app.js", "i18n.js", "styles.css",
		"manifest.webmanifest", "sw.js",
		"icons/icon-192.png", "icons/icon-512.png", "icons/maskable-512.png",
		"icons/apple-touch-icon.png",
	} {
		if _, err := fs.Stat(assets, name); err != nil {
			t.Errorf("missing frontend asset %q: %v", name, err)
		}
	}
}

// languageBlock returns the source text of one language object in i18n.js.
func languageBlock(t *testing.T, dictionary, language string) string {
	t.Helper()

	marker := "\n  " + language + ": {"
	start := strings.Index(dictionary, marker)
	if start < 0 {
		t.Fatalf("language block %q not found in i18n.js", language)
	}
	rest := dictionary[start+len(marker):]
	end := strings.Index(rest, "\n  }")
	if end < 0 {
		t.Fatalf("language block %q is not terminated", language)
	}
	return rest[:end]
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
