package handler

import (
	"encoding/xml"
	"strings"
	"testing"
)

// buildImageSitemap mirrors the markup HandleSitemapImages emits for one row so
// we can assert well-formedness without standing up a database.
func buildImageSitemap(slug, imageURL, prompt string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">` + "\n")
	b.WriteString("  <url>\n    <loc>")
	b.WriteString(xmlEscape(artSiteBase + "/art/i/" + slug))
	b.WriteString("</loc>\n    <changefreq>monthly</changefreq>\n")
	b.WriteString("    <image:image>\n      <image:loc>")
	b.WriteString(xmlEscape(imageURL))
	b.WriteString("</image:loc>\n      <image:caption>")
	b.WriteString(xmlEscape(artPromptCaption(prompt, 200)))
	b.WriteString("</image:caption>\n    </image:image>\n")
	b.WriteString("  </url>\n")
	b.WriteString("</urlset>\n")
	return b.String()
}

func TestArtSitemapIndexUsesCleanPathScheme(t *testing.T) {
	idx := ArtSitemapIndexXML("https://openpaths.io", 3)

	if strings.Contains(idx, "?p=") {
		t.Fatalf("sitemap index still uses query-string scheme:\n%s", idx)
	}
	for _, want := range []string{
		"<loc>https://openpaths.io/sitemap-pages.xml</loc>",
		"<loc>https://openpaths.io/sitemap-art-tags.xml</loc>",
		"<loc>https://openpaths.io/sitemap-art-1.xml</loc>",
		"<loc>https://openpaths.io/sitemap-art-3.xml</loc>",
	} {
		if !strings.Contains(idx, want) {
			t.Fatalf("index missing %q:\n%s", want, idx)
		}
	}
	if err := xml.Unmarshal([]byte(idx), new(struct {
		XMLName xml.Name `xml:"sitemapindex"`
	})); err != nil {
		t.Fatalf("sitemap index is not well-formed XML: %v", err)
	}
}

func TestStripInvalidXMLChars(t *testing.T) {
	// \x08 (backspace) is the exact byte that broke sitemap-art.xml in production.
	got := stripInvalidXMLChars("Greg Rutkowski and \x08Moebius")
	if want := "Greg Rutkowski and Moebius"; got != want {
		t.Fatalf("stripInvalidXMLChars = %q, want %q", got, want)
	}
	// Allowed whitespace and normal/unicode/emoji text must survive untouched.
	keep := "line1\nline2\ttab — café 🎨"
	if got := stripInvalidXMLChars(keep); got != keep {
		t.Fatalf("stripInvalidXMLChars mutated valid text: %q", got)
	}
}

func TestImageSitemapIsWellFormed(t *testing.T) {
	// A prompt loaded with everything that has historically broken the feed:
	// a forbidden control char, raw XML metacharacters, quotes and an emoji.
	prompt := "portrait <lady> & \"friend\" by Greg Rutkowski and \x08Moebius, 50% 'epic'  🎨"
	doc := buildImageSitemap("epic-portrait-00000ebf", "https://cutedsl.cc/a&b.webp", prompt)

	if strings.ContainsRune(doc, '\x08') {
		t.Fatal("sitemap still contains the forbidden \\x08 control character")
	}
	if err := xml.Unmarshal([]byte(doc), new(struct {
		XMLName xml.Name `xml:"urlset"`
	})); err != nil {
		t.Fatalf("sitemap is not well-formed XML: %v", err)
	}
}
