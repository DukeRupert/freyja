package email

import (
	"strings"
	"testing"
)

func TestGeneratePlainText(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string
		excludes []string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello, World!</p>",
			contains: []string{"Hello, World!"},
			excludes: []string{"<p>", "</p>"},
		},
		{
			name:     "line breaks",
			html:     "Line 1<br>Line 2<br/>Line 3<br />Line 4",
			contains: []string{"Line 1", "Line 2", "Line 3", "Line 4"},
			excludes: []string{"<br>", "<br/>", "<br />"},
		},
		{
			name:     "headings",
			html:     "<h1>Title</h1><h2>Subtitle</h2><h3>Section</h3>",
			contains: []string{"Title", "Subtitle", "Section"},
			excludes: []string{"<h1>", "</h1>", "<h2>", "</h2>", "<h3>", "</h3>"},
		},
		{
			name:     "nested tags",
			html:     "<div><p><strong>Bold text</strong> and <em>italic</em></p></div>",
			contains: []string{"Bold text", "and", "italic"},
			excludes: []string{"<div>", "<p>", "<strong>", "<em>"},
		},
		{
			name:     "HTML entities",
			html:     "Price: $10 &amp; shipping &nbsp; included &lt;$5&gt; &quot;free&quot;",
			contains: []string{"Price: $10 & shipping", "included <$5>", "\"free\""},
			excludes: []string{"&amp;", "&nbsp;", "&lt;", "&gt;", "&quot;"},
		},
		{
			name:     "links stripped",
			html:     `<a href="https://example.com">Click here</a>`,
			contains: []string{"Click here"},
			excludes: []string{"<a", "href", "</a>"},
		},
		{
			name:     "empty content",
			html:     "",
			contains: []string{},
			excludes: []string{},
		},
		{
			name: "email template structure",
			html: `
				<div class="email-content">
					<h2>Welcome!</h2>
					<p>Thank you for signing up.</p>
					<p>Click <a href="https://example.com/verify">here</a> to verify.</p>
				</div>
			`,
			contains: []string{"Welcome!", "Thank you for signing up", "here", "to verify"},
			excludes: []string{"<div", "<h2>", "<p>", "<a href"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePlainText(tt.html)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("generatePlainText() result should contain %q, got: %q", want, result)
				}
			}

			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("generatePlainText() result should not contain %q, got: %q", exclude, result)
				}
			}
		})
	}
}

func TestGeneratePlainText_WhitespaceHandling(t *testing.T) {
	html := `
		<p>   Line with spaces   </p>
		<p></p>
		<p>Another line</p>
	`

	result := generatePlainText(html)

	// Should not have empty lines (they get filtered)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" && line != "" {
			t.Error("generatePlainText() should not have blank lines with only whitespace")
		}
	}

	// Should contain the actual content
	if !strings.Contains(result, "Line with spaces") {
		t.Error("generatePlainText() should contain trimmed content")
	}
	if !strings.Contains(result, "Another line") {
		t.Error("generatePlainText() should contain 'Another line'")
	}
}
