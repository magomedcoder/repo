package seo

import (
	"encoding/json"
	"html"
	"strings"
)

type Meta struct {
	Status      int
	Title       string
	Description string
	Canonical   string
	Robots      string
	OGType      string
	OGLocale    string
	OGImage     string
	SiteName    string
	JSONLD      []map[string]any
}

func (m Meta) robots() string {
	if m.Robots != "" {
		return m.Robots
	}

	return "index,follow"
}

func (m Meta) ogType() string {
	if m.OGType != "" {
		return m.OGType
	}

	return "website"
}

func (m Meta) jsonLDBytes() []byte {
	if len(m.JSONLD) == 0 {
		return nil
	}

	var payload any = m.JSONLD[0]
	if len(m.JSONLD) > 1 {
		payload = m.JSONLD
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	return b
}

func InjectHTML(raw []byte, m Meta) []byte {
	htmlStr := string(raw)
	title := html.EscapeString(strings.TrimSpace(m.Title))
	if title == "" {
		title = html.EscapeString(m.SiteName)
	}

	if strings.Contains(htmlStr, "<title>") {
		start := strings.Index(htmlStr, "<title>")
		end := strings.Index(htmlStr, "</title>")
		if start >= 0 && end > start {
			htmlStr = htmlStr[:start] + "<title>" + title + "</title>" + htmlStr[end+len("</title>"):]
		}
	} else {
		htmlStr = strings.Replace(htmlStr, "<head>", "<head>\n    <title>"+title+"</title>", 1)
	}

	var b strings.Builder
	b.WriteString("\n    <meta name=\"description\" content=\"")
	b.WriteString(html.EscapeString(trimMeta(m.Description, 300)))
	b.WriteString("\" />\n")
	b.WriteString("    <meta name=\"robots\" content=\"")
	b.WriteString(html.EscapeString(m.robots()))
	b.WriteString("\" />\n")
	if m.Canonical != "" {
		b.WriteString("    <link rel=\"canonical\" href=\"")
		b.WriteString(html.EscapeString(m.Canonical))
		b.WriteString("\" />\n")
	}
	b.WriteString("    <meta property=\"og:type\" content=\"")
	b.WriteString(html.EscapeString(m.ogType()))
	b.WriteString("\" />\n")
	b.WriteString("    <meta property=\"og:title\" content=\"")
	b.WriteString(title)
	b.WriteString("\" />\n")
	b.WriteString("    <meta property=\"og:description\" content=\"")
	b.WriteString(html.EscapeString(trimMeta(m.Description, 300)))
	b.WriteString("\" />\n")
	if m.Canonical != "" {
		b.WriteString("    <meta property=\"og:url\" content=\"")
		b.WriteString(html.EscapeString(m.Canonical))
		b.WriteString("\" />\n")
	}
	if m.SiteName != "" {
		b.WriteString("    <meta property=\"og:site_name\" content=\"")
		b.WriteString(html.EscapeString(m.SiteName))
		b.WriteString("\" />\n")
	}
	if m.OGLocale != "" {
		b.WriteString("    <meta property=\"og:locale\" content=\"")
		b.WriteString(html.EscapeString(m.OGLocale))
		b.WriteString("\" />\n")
	}
	if m.OGImage != "" {
		b.WriteString("    <meta property=\"og:image\" content=\"")
		b.WriteString(html.EscapeString(m.OGImage))
		b.WriteString("\" />\n")
	}
	b.WriteString("    <meta name=\"twitter:card\" content=\"summary\" />\n")
	b.WriteString("    <meta name=\"twitter:title\" content=\"")
	b.WriteString(title)
	b.WriteString("\" />\n")
	b.WriteString("    <meta name=\"twitter:description\" content=\"")
	b.WriteString(html.EscapeString(trimMeta(m.Description, 300)))
	b.WriteString("\" />\n")
	if ld := m.jsonLDBytes(); len(ld) > 0 {
		b.WriteString("    <script type=\"application/ld+json\">")
		b.Write(ld)
		b.WriteString("</script>\n")
	}

	insert := b.String()
	if i := strings.Index(htmlStr, "</head>"); i >= 0 {
		htmlStr = htmlStr[:i] + insert + htmlStr[i:]
	}

	return []byte(htmlStr)
}

func trimMeta(s string, max int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if max <= 0 || len(s) <= max {
		return s
	}

	if max < 3 {
		return s[:max]
	}

	return strings.TrimSpace(s[:max-1]) + "..."
}
