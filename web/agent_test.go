package web

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The standalone agent app lives in web/agent and is copied verbatim into the
// Android APK assets, so both copies must stay identical and internally
// consistent.

func TestAgentSPASelectorsExist(t *testing.T) {
	html := readFrom(t, AgentFS(), "index.html")
	script := readFrom(t, AgentFS(), "agent.js")

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
		t.Fatalf("agent.js references elements missing from index.html: %v", sortedKeys(missing))
	}
}

func TestAgentTranslationKeysAreComplete(t *testing.T) {
	html := readFrom(t, AgentFS(), "index.html")
	script := readFrom(t, AgentFS(), "agent.js")
	dictionary := readFrom(t, AgentFS(), "i18n.js")

	languages := map[string]map[string]bool{}
	for _, language := range []string{"ar", "en"} {
		keys := map[string]bool{}
		for _, match := range keyInDict.FindAllStringSubmatch(languageBlock(t, dictionary, language), -1) {
			keys[match[1]] = true
		}
		languages[language] = keys
	}

	used := map[string]bool{}
	for _, match := range keyInHTML.FindAllStringSubmatch(html, -1) {
		used[match[1]] = true
	}
	for _, match := range keyInJS.FindAllStringSubmatch(script, -1) {
		if !strings.HasSuffix(match[1], ".") {
			used[match[1]] = true
		}
	}
	// Suggestion buttons carry their translation key in data-suggestion.
	for _, match := range suggestionInHTML.FindAllStringSubmatch(html, -1) {
		used[match[1]] = true
	}

	for language, keys := range languages {
		for key := range used {
			if !keys[key] {
				t.Errorf("agent translation %q is missing from the %s dictionary", key, language)
			}
		}
	}
	for key := range languages["ar"] {
		if !languages["en"][key] {
			t.Errorf("agent key %q exists in ar but not in en", key)
		}
	}
}

func TestAgentAssetsMatchAndroidCopy(t *testing.T) {
	const androidRoot = "../android/app/src/main/assets/agent"
	if _, err := os.Stat(androidRoot); err != nil {
		t.Skipf("android assets not present: %v", err)
	}

	err := fs.WalkDir(AgentFS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		expected, readErr := fs.ReadFile(AgentFS(), name)
		if readErr != nil {
			return readErr
		}
		actual, readErr := os.ReadFile(filepath.Join(androidRoot, filepath.FromSlash(name)))
		if readErr != nil {
			t.Errorf("missing in android assets: %s (run: go run ./tools/syncassets)", name)
			return nil
		}
		if !bytes.Equal(expected, actual) {
			t.Errorf("out of sync: %s (run: go run ./tools/syncassets)", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk agent assets: %v", err)
	}
}
